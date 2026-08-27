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

    # Redis Connection Settings (Distributed Rate Limiting)
    REDIS_HOST: str = "redis"
    REDIS_PORT: int = 6379
    REDIS_PASSWORD: str = ""
    REDIS_DB: int = 0

    # Ollama Local LLM Settings (Qwen 2.5)
    OLLAMA_BASE_URL: str = "http://ollama:11434"
    LLM_MODEL: str = "qwen2.5:3b"

    # LLM Temperature Settings
    LLM_TOOL_TEMPERATURE: float = 0.0
    LLM_CHAT_TEMPERATURE: float = 0.3

    # Similarity & Search Accuracy Thresholds (Calibrated for multilingual embeddings)
    DEFAULT_SEARCH_SCORE_THRESHOLD: float = 0.25
    CHAT_SEARCH_SCORE_THRESHOLD: float = 0.20
    CHAT_SEARCH_FALLBACK_THRESHOLD: float = 0.10



    # Hybrid Search Settings (Dense Vector + Trigram Text Matching)
    ENABLE_HYBRID_SEARCH: bool = True
    HYBRID_VECTOR_WEIGHT: float = 0.70
    HYBRID_TEXT_WEIGHT: float = 0.30

    # Search Limit Defaults
    SEARCH_LIMIT: int = 12
    CHAT_SEARCH_LIMIT: int = 10

    # API Security Settings
    CORS_ORIGINS: str = "http://localhost:3000,http://127.0.0.1:3000,*"
    RATE_LIMIT_ENABLED: bool = True
    RATE_LIMIT_GENERAL_PER_MINUTE: int = 60
    RATE_LIMIT_CHAT_PER_MINUTE: int = 30
    MAX_REQUEST_BODY_BYTES: int = 2_097_152  # 2MB
    INTERNAL_API_KEY: str = ""

    class Config:
        env_file = ".env"
        extra = "ignore"


settings = Settings()

