from locust import HttpUser, task, between
import random
from datetime import datetime, timezone

class MilitaryTraffic(HttpUser):
    # Tiempo de espera entre peticiones de cada usuario simulado
    wait_time = between(1, 3)

    @task
    def send_report(self):
        countries = ["USA", "RUS", "CHN", "ESP", "GTM"]
        payload = {
            "country": random.choice(countries),
            "warplanes_in_air": random.randint(0, 50),
            "warships_in_water": random.randint(0, 30),
            "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        }
        
        # Enviar petición al endpoint expuesto por Envoy Gateway
        self.client.post("/grpc-201907078", json=payload)