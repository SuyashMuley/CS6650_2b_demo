import os
import random

from locust import HttpUser, between, task

# LOCUST_CLIENT=fast switches to FastHttpUser (geventhttpclient); default is HttpUser (requests).
if os.getenv("LOCUST_CLIENT", "http").lower() == "fast":
    from locust.contrib.fasthttp import FastHttpUser as BaseUser
else:
    BaseUser = HttpUser

# The server seeds products 1-3 at startup.
SEEDED_IDS = [1, 2, 3]
MISSING_ID = 999999

# LOCUST_WAIT=1 adds think time (0.1-0.5s) between requests; default is back-to-back.
WAIT = between(0.1, 0.5) if os.getenv("LOCUST_WAIT") == "1" else None


class ProductUser(BaseUser):
    if WAIT:
        wait_time = WAIT

    # Read-heavy mix, as in a real store: ~80% hits, ~10% misses, ~10% updates.
    @task(8)
    def get_product(self):
        self.client.get(f"/products/{random.choice(SEEDED_IDS)}", name="GET /products/{id}")

    @task(1)
    def get_missing_product(self):
        # 404 is the expected answer here, so don't count it as a failure.
        with self.client.get(
            f"/products/{MISSING_ID}", name="GET /products/{id} (404)", catch_response=True
        ) as resp:
            if resp.status_code == 404:
                resp.success()

    @task(1)
    def update_product(self):
        pid = random.choice(SEEDED_IDS)
        # 204 is the success code for this endpoint.
        self.client.post(
            f"/products/{pid}/details",
            json={
                "product_id": pid,
                "sku": f"LOAD-{random.randint(1, 9999)}",
                "manufacturer": "Locust",
                "category_id": random.randint(1, 50),
                "weight": random.randint(0, 5000),
                "some_other_id": random.randint(1, 100),
            },
            name="POST /products/{id}/details",
        )
