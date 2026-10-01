"""Carga básica del flujo principal; evaluación de escalado corresponde a fase 6."""
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import random
from uuid import uuid4
from locust import HttpUser, between, task

PROFILE = json.loads(Path(os.environ.get("LOAD_PROFILE", "/load/load-profile.json")).read_text())
if not set(PROFILE["countries"]) <= {"USA", "RUS", "CHN", "ESP", "GTM"} or not PROFILE["countries"]:
    raise ValueError("países inválidos en perfil")
for name, limit in (("warplanes_in_air", 50), ("warships_in_water", 30)):
    low, high = PROFILE[name]
    if type(low) is not int or type(high) is not int or not 0 <= low <= high <= limit:
        raise ValueError("rango inválido: " + name)
RUN_ID = os.environ.get("RUN_ID", "locust-" + str(uuid4()))


class Reports(HttpUser):
    wait_time = between(0.5, 1.5)

    @task
    def report(self):
        event_id = str(uuid4())
        payload = {
            "country": random.choice(PROFILE["countries"]),
            "warplanes_in_air": random.randint(*PROFILE["warplanes_in_air"]),
            "warships_in_water": random.randint(*PROFILE["warships_in_water"]),
            "timestamp": datetime.now(timezone.utc).isoformat(timespec="microseconds").replace("+00:00", "Z"),
        }
        with self.client.post("/grpc-202100054", json=payload,
                              headers={"Idempotency-Key": event_id, "X-Run-ID": RUN_ID},
                              timeout=7, catch_response=True) as response:
            if response.status_code != 202:
                response.failure("No aceptado: " + str(response.status_code))
            else:
                body = response.json()
                if body.get("event_id") != event_id or body.get("run_id") != RUN_ID:
                    response.failure("Identidad de confirmación incorrecta")
