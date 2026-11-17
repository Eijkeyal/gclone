from fastapi import FastAPI


def create_app() -> FastAPI:
    app = FastAPI()

    @app.get("/")
    def index():
        return {"status": "factory ready"}

    return app
