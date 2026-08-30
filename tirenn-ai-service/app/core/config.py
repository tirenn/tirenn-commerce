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

    # Ollama Local LLM Settings (Qwen 2.5 3B for intelligent Tool Calling)
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
    CHAT_SEARCH_LIMIT: int = 6

    # API Security Settings
    CORS_ORIGINS: str = "http://localhost:3000,http://127.0.0.1:3000,*"
    RATE_LIMIT_ENABLED: bool = True
    RATE_LIMIT_GENERAL_PER_MINUTE: int = 300
    RATE_LIMIT_CHAT_PER_MINUTE: int = 180
    MAX_REQUEST_BODY_BYTES: int = 10_485_760  # 10MB

    # JWT Authentication Settings (Synced with Golang Backend)
    JWT_SECRET: str = "super-secret-tirenn-jwt-key-2026"
    JWT_ISSUER: str = "gocommerce-api"

    # Redis Session History & Sliding Window Settings
    SESSION_HISTORY_LIMIT: int = 10  # Number of past messages fetched for LLM context window
    SESSION_MAX_STORED: int = 50     # Max messages retained in Redis List (via LTRIM)
    SESSION_TTL_SECONDS: int = 86400 # 24-hour expiration for inactive chat sessions

    # Redis RAG Semantic Cache Settings
    RAG_CACHE_ENABLED: bool = True
    RAG_CACHE_SEMANTIC_THRESHOLD: float = 0.92  # 92% similarity threshold for semantic cache hit
    RAG_CACHE_TTL_SECONDS: int = 86400          # 24-hour expiration for cached RAG responses
    RAG_CACHE_MAX_ENTRIES: int = 100            # Max semantic vector entries stored per document scope

    # Internal Machine-to-Machine Secret Key
    INTERNAL_API_KEY: str = "very-very-secret-internal-key-2026"

    class Config:
        env_file = ".env"
        extra = "ignore"


settings = Settings()

