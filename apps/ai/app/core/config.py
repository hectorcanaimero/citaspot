# Configuración centralizada del AI Service — cargada desde variables de entorno.
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    app_name: str = "CitaSpot - AI Service"
    app_env: str = "development"
    debug: bool = False
    database_url: str = ""
    redis_url: str = "redis://localhost:6379"
    rabbitmq_url: str = "amqp://guest:guest@localhost:5672/"
    core_api_url: str = "http://localhost:3001"
    gemini_api_key: str = ""
    openai_api_key: str = ""
    deepseek_api_key: str = ""

    model_config = {"env_file": ".env", "case_sensitive": False}


settings = Settings()
