import os
from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    SERVICE_NAME: str = "Tirenn AI Commerce & Shopper Service"
    PORT: int = 8000
    HOST: str = "0.0.0.0"
    ENVIRONMENT: str = "development"
    
    # SOTA Multilingual Embedding Model via Ollama (e.g., bge-m3, paraphrase-multilingual, nomic-embed-text)
    EMBEDDING_MODEL_NAME: str = "bge-m3"
    
    # PostgreSQL & pgvector Connection Settings
    DB_HOST: str = "postgres"
    DB_PORT: int = 5432
    DB_USER: str = "postgres_user111"
    DB_PASSWORD: str = ""
    DB_NAME: str = "commerce_db"
    
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
    LLM_NUM_PREDICT: int = 350
    LLM_NUM_CTX: int = 2048
    LLM_KEEP_ALIVE: str = "60m"
    LLM_TIMEOUT: float = 120.0
    MAX_AGENT_ITERATIONS: int = 5

    # LLM Temperature Settings
    LLM_TOOL_TEMPERATURE: float = 0.0
    LLM_CHAT_TEMPERATURE: float = 0.3

    # Embedding Settings
    EMBEDDING_DIMENSIONS: int = 1024

    # Similarity & Search Accuracy Thresholds (Calibrated for multilingual embeddings)
    DEFAULT_SEARCH_SCORE_THRESHOLD: float = 0.38
    CHAT_SEARCH_SCORE_THRESHOLD: float = 0.30
    CHAT_SEARCH_FALLBACK_THRESHOLD: float = 0.20
    RECOMMENDATION_SIMILARITY_THRESHOLD: float = 0.35

    # Hybrid Search Settings (Dense Vector + Trigram Text Matching)
    ENABLE_HYBRID_SEARCH: bool = True
    HYBRID_VECTOR_WEIGHT: float = 0.70
    HYBRID_TEXT_WEIGHT: float = 0.30

    # Search Limit Defaults
    SEARCH_LIMIT: int = 12
    CHAT_SEARCH_LIMIT: int = 6
    RECOMMENDATIONS_LIMIT: int = 6

    # API Security Settings
    CORS_ORIGINS: str = "http://localhost:3000,http://127.0.0.1:3000,*"
    RATE_LIMIT_ENABLED: bool = True
    RATE_LIMIT_GENERAL_PER_MINUTE: int = 300
    RATE_LIMIT_CHAT_PER_MINUTE: int = 180
    MAX_REQUEST_BODY_BYTES: int = 10_485_760  # 10MB

    # JWT Authentication Settings (Synced with Golang Backend)
    JWT_SECRET: str = ""
    JWT_ISSUER: str = "commerce-api"

    # Redis Session History & Sliding Window Settings
    SESSION_HISTORY_LIMIT: int = 10  # Number of past messages fetched for LLM context window
    SESSION_MAX_STORED: int = 50     # Max messages retained in Redis List (via LTRIM)
    SESSION_TTL_SECONDS: int = 86400 # 24-hour expiration for inactive chat sessions

    # Redis RAG Semantic Cache Settings
    RAG_CACHE_ENABLED: bool = True
    RAG_CACHE_SEMANTIC_THRESHOLD: float = 0.92  # 92% similarity threshold for semantic cache hit
    RAG_CACHE_TTL_SECONDS: int = 86400          # 24-hour expiration for cached RAG responses
    RAG_CACHE_MAX_ENTRIES: int = 100            # Max semantic vector entries stored per document scope

    # Redis LLM Response Semantic Cache Settings (GPTCache Pattern)
    LLM_CACHE_ENABLED: bool = True
    LLM_CACHE_SEMANTIC_THRESHOLD: float = 0.92  # 92% vector similarity threshold for cache hit
    LLM_CACHE_EXACT_TTL_SECONDS: int = 7200     # 2-hour TTL for exact hash cached chat replies
    LLM_CACHE_SEMANTIC_TTL_SECONDS: int = 7200  # 2-hour TTL for semantic cached chat replies
    LLM_CACHE_MAX_ENTRIES: int = 500            # Max cached conversational turns per domain scope

    # Internal Machine-to-Machine Secret Key
    INTERNAL_API_KEY: str = ""


settings = Settings()

