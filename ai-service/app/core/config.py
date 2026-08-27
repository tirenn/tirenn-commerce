import os
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    SERVICE_NAME: str = "Tirenn AI Commerce & Shopper Service"
    PORT: int = 8000
    HOST: str = "0.0.0.0"
    ENVIRONMENT: str = "development"
    
    # Embedding Model (BAAI/bge-small-en-v1.5 or BAAI/bge-m3 for multilingual)
    EMBEDDING_MODEL_NAME: str = "BAAI/bge-small-en-v1.5"
    
    # Qdrant Vector Storage Path (None = in-memory)
    QDRANT_STORAGE_PATH: str = "./data/qdrant_storage"
    
    # Core Backend URL
    BACKEND_API_URL: str = "http://localhost:8080/api/v1"

    # Ollama Local LLM Settings (Qwen 2.5)
    OLLAMA_BASE_URL: str = "http://localhost:11434"
    LLM_MODEL: str = "qwen2.5:3b"

    class Config:
        env_file = ".env"
        extra = "ignore"

settings = Settings()
