import os
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    SERVICE_NAME: str = "Tirenn AI Commerce & Shopper Service"
    PORT: int = 8000
    HOST: str = "0.0.0.0"
    ENVIRONMENT: str = "development"
    
    # SOTA Multilingual & Indonesian Embedding Model (384 dimensions, ~220MB)
    EMBEDDING_MODEL_NAME: str = "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2"
    
    # PostgreSQL & pgvector Connection Settings
    DB_HOST: str = "postgres"
    DB_PORT: int = 5432
    DB_USER: str = "gouser"
    DB_PASSWORD: str = "gopassword"
    DB_NAME: str = "gocommerce_db"
    
    # Core Backend URL
    BACKEND_API_URL: str = "http://backend:8080/api/v1"

    # Ollama Local LLM Settings (Qwen 2.5)
    OLLAMA_BASE_URL: str = "http://ollama:11434"
    LLM_MODEL: str = "qwen2.5:3b"

    class Config:
        env_file = ".env"
        extra = "ignore"

settings = Settings()
