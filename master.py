from locust import HttpUser, task


class LocustMaster(HttpUser):
    @task
    def check_health(self):
        pass
