#!/usr/bin/env python3
"""Pruebas reales sobre el stack LOCAL mumn-p3; no usa el Proyecto 2 ni GCP."""
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
from copy import deepcopy
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import time
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen
from uuid import uuid4

ROOT = Path(__file__).resolve().parents[1]
COMPOSE = ["docker", "compose", "-f", str(ROOT / "deploy/compose.local.yaml")]
URL = "http://127.0.0.1:" + os.environ.get("MUMN_PORT", "18080")
spec = importlib.util.spec_from_file_location("reference", ROOT / "scripts/check.py")
reference = importlib.util.module_from_spec(spec)
spec.loader.exec_module(reference)


def compose(*args):
    return subprocess.check_output(COMPOSE + list(args), text=True, stderr=subprocess.STDOUT)


def valkey(*args):
    return json.loads(compose("exec", "-T", "valkey", "valkey-cli", "--json", *args))


def state(run):
    result = valkey("HGETALL", "mumn:run:" + run)
    return result if isinstance(result, dict) else dict(zip(result[::2], result[1::2]))


def wait_for(fn, timeout=40):
    end = time.monotonic() + timeout
    last = None
    while time.monotonic() < end:
        try:
            result = fn()
            if result:
                return result
        except (OSError, ValueError, subprocess.CalledProcessError) as error:
            last = error
        time.sleep(0.3)
    raise AssertionError(f"Condición no satisfecha en {timeout}s; último error: {last}")


def post(report, run, event_id=None, content_type="application/json"):
    body = report if isinstance(report, bytes) else json.dumps(report).encode()
    headers = {"Content-Type": content_type, "X-Run-ID": run}
    if event_id is not None:
        headers["Idempotency-Key"] = event_id
    request = Request(URL + "/grpc-202100054", data=body, headers=headers)
    try:
        with urlopen(request, timeout=8) as response:
            return response.status, json.load(response)
    except HTTPError as error:
        return error.code, error.read().decode()


def drain():
    def empty():
        rows = json.loads(compose("exec", "-T", "rabbitmq", "rabbitmqctl", "list_queues", "name", "messages", "--formatter", "json", "-q"))
        return any(row["name"] == "mumn.reports.v1" and row["messages"] == 0 for row in rows)
    wait_for(empty)


def assert_stats(run, expected):
    values = state(run)
    events = [{"event_id": name[6:], "report": json.loads(value)} for name, value in values.items() if name.startswith("event:")]
    actual = reference.aggregate(events)
    assert actual == expected, (actual, expected)
    assert int(values["count"]) == expected["total"]
    assert int(values.get("country:CHN:count", "0")) == expected["country_count"]
    for field, prefix in (("warplanes_in_air", "planes"), ("warships_in_water", "ships")):
        histogram = Counter(event["report"][field] for event in events)
        saved = {int(k.split(":")[1]): int(v) for k, v in values.items() if k.startswith(prefix + ":")}
        assert saved == histogram, (saved, histogram)
    latest = {}
    for event in sorted(events, key=lambda e: (reference.timestamp(e["report"]["timestamp"]), e["event_id"])):
        latest[event["report"]["country"]] = event["report"]
    for country, report in latest.items():
        assert json.loads(values["country:" + country + ":latest"]) == report


def main():
    fixture = json.loads((ROOT / "tests/fixtures/reports.json").read_text())
    expected = json.loads((ROOT / "tests/fixtures/expected.json").read_text())
    run = "integration-" + str(uuid4())
    results = []
    def passed(label):
        results.append(label)
        print("OK:", label, flush=True)
    sample = fixture[0]["report"]
    wait_for(lambda: urlopen(URL + "/healthz", timeout=2).status == 200)
    # Health is process-only. Probe the whole chain before sending fixture data.
    wait_for(lambda: post(sample, run + "-probe", fixture[0]["event_id"])[0] == 202, 90)
    for event in fixture:
        code, response = post(event["report"], run, event["event_id"])
        assert code == 202 and response["event_id"] == event["event_id"], response
    wait_for(lambda: state(run).get("count") == "5")
    assert_stats(run, expected)
    passed("Rust → REST Go → gRPC → writer → RabbitMQ → Valkey: dataset exacto")

    with ThreadPoolExecutor(max_workers=12) as pool:
        responses = list(pool.map(lambda _: post(fixture[2]["report"], run, fixture[2]["event_id"]), range(24)))
    assert all(response[0] == 202 for response in responses), responses
    equivalent = deepcopy(sample)
    equivalent["timestamp"] = "2026-09-29T06:00:00-06:00"
    assert post(equivalent, run, fixture[0]["event_id"])[0] == 202
    drain()
    assert_stats(run, expected)
    passed("24 reentregas concurrentes y zona horaria equivalente no duplican datos")

    new = deepcopy(fixture[2]); new["event_id"] = str(uuid4())
    assert post(new["report"], run, new["event_id"])[0] == 202
    wait_for(lambda: state(run).get("count") == "6")
    assert_stats(run, reference.aggregate(fixture + [new]))
    passed("Contenido idéntico con UUID nuevo cuenta como reporte nuevo")

    before_failed = json.loads(compose("exec", "-T", "rabbitmq", "rabbitmqctl", "list_queues", "name", "messages", "--formatter", "json", "-q"))
    before_failed = next(q["messages"] for q in before_failed if q["name"] == "mumn.reports.failed")
    conflict = deepcopy(sample); conflict["warplanes_in_air"] = 49
    assert post(conflict, run, fixture[0]["event_id"])[0] == 202
    drain()
    assert_stats(run, reference.aggregate(fixture + [new]))
    failed = json.loads(compose("exec", "-T", "rabbitmq", "rabbitmqctl", "list_queues", "name", "messages", "--formatter", "json", "-q"))
    assert any(q["name"] == "mumn.reports.failed" and q["messages"] == before_failed + 1 for q in failed)
    passed("Conflicto de identidad se conserva en cola de fallos sin sobrescribir")

    late_run = run + "-late"
    for event in (fixture[4], fixture[0]):
        assert post(event["report"], late_run, event["event_id"])[0] == 202
    wait_for(lambda: state(late_run).get("count") == "2")
    assert_stats(late_run, reference.aggregate([fixture[4], fixture[0]]))
    passed("Evento tardío no sustituye el último estado del país")

    for field, value in (("country", "xxx"), ("warplanes_in_air", True), ("warplanes_in_air", 10.0), ("warships_in_water", 31), ("timestamp", "2026-02-30T12:00:00Z")):
        invalid = deepcopy(sample); invalid[field] = value
        assert post(invalid, run)[0] == 400, (field, value)
    assert post(sample, run, "bad-uuid")[0] == 400
    assert post(sample, "bad/run")[0] == 400
    assert post(sample, run, content_type="text/plain")[0] == 415
    assert post(b" " * 5000, run)[0] == 413
    passed("API rechaza tipos, rangos, fechas, cabeceras y cuerpos fuera del contrato")

    recovery_run = run + "-recover"
    compose("stop", "valkey")
    try:
        assert post(sample, recovery_run, fixture[0]["event_id"])[0] == 202
        time.sleep(1)
        compose("restart", "consumer")
    finally:
        compose("start", "valkey")
    wait_for(lambda: state(recovery_run).get("count") == "1")
    passed("Valkey caído y consumidor reiniciado: mensaje sin ACK se recupera")

    compose("restart", "consumer", "valkey")
    wait_for(lambda: state(run).get("count") == "6")
    assert_stats(run, reference.aggregate(fixture + [new]))
    passed("Reinicio de consumidor y Valkey conserva los datos del volumen")

    compose("stop", "rabbitmq")
    try:
        start = time.monotonic()
        code, _ = post(sample, run + "-unavailable")
        assert code in (503, 504), code
        assert time.monotonic() - start < 7
    finally:
        compose("start", "rabbitmq")
    wait_for(lambda: post(sample, run + "-restored", fixture[0]["event_id"])[0] == 202, 90)
    wait_for(lambda: state(run + "-restored").get("count") == "1")
    passed("Broker caído no produce éxito falso; publicador y consumidor reconectan")
    artifact = ROOT / "artifacts/integration-latest.json"
    artifact.parent.mkdir(exist_ok=True)
    artifact.write_text(json.dumps({"run_id": run, "passed": results}, ensure_ascii=False, indent=2) + "\n")
    print(f"{len(results)} escenarios aprobados. Evidencia local: {artifact}")


if __name__ == "__main__":
    main()
