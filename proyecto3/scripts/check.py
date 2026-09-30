#!/usr/bin/env python3
"""Comprueba el esqueleto y los datos de referencia; no ejecuta servicios."""

import argparse
from collections import Counter
from datetime import datetime, timezone
import json
from pathlib import Path
import re
import sys
from uuid import UUID

ROOT = Path(__file__).resolve().parents[1]
COUNTRIES = ("USA", "RUS", "CHN", "ESP", "GTM")
FIELDS = {"country", "warplanes_in_air", "warships_in_water", "timestamp"}


def timestamp(value):
    if not isinstance(value, str) or not re.fullmatch(
        r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?(?:Z|[+-]\d{2}:\d{2})", value
    ):
        raise ValueError("timestamp requiere fecha y zona RFC 3339, con hasta seis decimales")
    if value[-1] != "Z":
        hours, minutes = map(int, value[-5:].split(":"))
        if hours > 23 or minutes > 59:
            raise ValueError("zona horaria inválida")
    return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(timezone.utc)


def validate_report(report):
    if not isinstance(report, dict) or set(report) != FIELDS:
        raise ValueError("campos del reporte incorrectos")
    if report["country"] not in COUNTRIES:
        raise ValueError("país no admitido")
    for name, limit in (("warplanes_in_air", 50), ("warships_in_water", 30)):
        value = report[name]
        if type(value) is not int or not 0 <= value <= limit:
            raise ValueError(f"{name} requiere entero entre 0 y {limit}")
    timestamp(report["timestamp"])


def aggregate(events):
    """Oráculo local de referencia para comparar futuras implementaciones."""
    unique = {}
    for event in events:
        if set(event) != {"event_id", "report"}:
            raise ValueError("envoltura del fixture incorrecta")
        event_id = str(UUID(event["event_id"]))
        report = event["report"]
        validate_report(report)
        report = dict(report)
        report["timestamp"] = timestamp(report["timestamp"]).isoformat().replace("+00:00", "Z")
        if event_id in unique and unique[event_id] != report:
            raise ValueError("identidad repetida con contenido distinto")
        unique[event_id] = report
    ordered = sorted(unique.items(), key=lambda item: (timestamp(item[1]["timestamp"]), item[0]))
    latest = {}
    for _, report in ordered:
        latest[report["country"]] = report

    def stats(field):
        counts = Counter(report[field] for report in unique.values())
        if not counts:
            return {"min": None, "max": None, "modes": [], "frequency": 0}
        frequency = max(counts.values())
        return {"min": min(counts), "max": max(counts),
                "modes": sorted(v for v, n in counts.items() if n == frequency),
                "frequency": frequency}

    def ranking(field):
        values = [[country, report[field]] for country, report in latest.items()]
        return sorted(values, key=lambda item: (-item[1], item[0]))

    series = [[timestamp(report["timestamp"]).isoformat().replace("+00:00", "Z"),
               report["warplanes_in_air"], report["warships_in_water"]]
              for _, report in ordered if report["country"] == "CHN"]
    return {"total": len(unique), "country": "CHN", "country_count": len(series),
            "planes": stats("warplanes_in_air"), "ships": stats("warships_in_water"),
            "top_planes": ranking("warplanes_in_air"),
            "top_ships": ranking("warships_in_water"), "series": series}


def check_contracts():
    schema = json.loads((ROOT / "contracts/report.schema.json").read_text())
    props = schema["properties"]
    if (schema["type"] != "object" or schema["additionalProperties"] is not False
            or set(schema["required"]) != FIELDS or set(props) != FIELDS
            or props["country"] != {"type": "string", "enum": list(COUNTRIES)}
            or props["timestamp"] != {"type": "string", "format": "date-time"}):
        raise ValueError("el esquema no coincide con el contrato aprobado")
    for field, limit in (("warplanes_in_air", 50), ("warships_in_water", 30)):
        if props[field] != {"type": "integer", "minimum": 0, "maximum": limit}:
            raise ValueError(f"límites del esquema incorrectos: {field}")
    proto = (ROOT / "contracts/war_report.proto").read_text()
    required = ["Countries country = 1;", "int32 warplanes_in_air = 2;",
                "int32 warships_in_water = 3;", "string timestamp = 4;",
                "countries_unknown = 0;", "usa = 1;", "rus = 2;", "chn = 3;",
                "esp = 4;", "gtm = 5;",
                "rpc SendReport (WarReportRequest) returns (WarReportResponse);"]
    if any(token not in proto for token in required):
        raise ValueError("faltan declaraciones protobuf esperadas; compilar con protoc en fase 3")


def check_docs():
    for name in ("README.md", "contracts", "services/rust-api", "services/go", "operator",
                 "deploy", "infra", "load", "tests/fixtures", "tests/test_contracts.py",
                 "scripts/doctor.py", "ci/check.sh", ".githooks/pre-push", "Makefile", "docs/ramas.md"):
        if not (ROOT / name).exists():
            raise ValueError(f"falta estructura requerida: {name}")
    for path in ROOT.rglob("*.md"):
        content = path.read_text()
        if sum(line.startswith("```") for line in content.splitlines()) % 2:
            raise ValueError(f"bloque Markdown abierto: {path.relative_to(ROOT)}")
        for link in re.findall(r"\]\(([^)]+)\)", content):
            if "://" not in link and not link.startswith("#"):
                if not (path.parent / link.split("#")[0]).exists():
                    raise ValueError(f"enlace inexistente en {path.relative_to(ROOT)}: {link}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--expected", type=Path, default=ROOT / "tests/fixtures/expected.json")
    args = parser.parse_args()
    try:
        check_docs()
        check_contracts()
        actual = aggregate(json.loads((ROOT / "tests/fixtures/reports.json").read_text()))
        expected = json.loads(args.expected.read_text())
        if actual != expected:
            raise ValueError("estadísticas distintas del resultado esperado")
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(f"FALLO: {error}", file=sys.stderr)
        return 1
    print("OK: estructura, contratos acotados y dataset de referencia.")
    print("ALCANCE: no acredita compilación, servicios, integración, Operator ni GKE.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
