---
name: senior-ai-engineer
description: >-
  Provides end-to-end expertise in Generative AI, LLM systems, Agentic architectures (LangGraph, AutoGen, CrewAI),
  RAG pipelines, vector databases, prompt engineering, fine-tuning, LLM evaluation, and production AI engineering.
---

# Senior AI Engineer Skill & Agent Guide

This skill equips the agent with senior-level Artificial Intelligence and Machine Learning engineering expertise, with a deep focus on Generative AI, Large Language Models (LLMs), Agentic workflows, RAG architectures, model evaluation, and production deployment.

---

## 1. Core Competencies & Architecture Patterns

### A. Agentic Architectures & Multi-Agent Orchestration
- **Single & Multi-Agent Workflows**: State machines, graph-based execution (e.g., LangGraph, AutoGen, CrewAI), hierarchical supervisor-worker patterns, and autonomous loops (ReAct, Plan-and-Execute, Reflection/Self-Correction).
- **Tool Calling & Function Calling**: Strict schema validation with Pydantic / JSON Schema, defensive parameter coercion, and structured outputs.
- **Memory Management**: Short-term conversational context buffers, token-budgeted memory summarization, and long-term episodic/semantic memory retrieval.
- **Human-in-the-Loop (HITL)**: Interruptible graph states, review checkpoints, and action approvals for critical operations.

### B. Advanced Retrieval-Augmented Generation (RAG)
- **Ingestion & Preprocessing**:
  - Document parsing (PDF, Markdown, HTML, DOCX) with structure preservation (tables, hierarchy).
  - Chunking strategies: Recursive character splitting, semantic chunking, sliding window with overlap, parent-document / hierarchical chunking.
- **Vector Indexing & Hybrid Search**:
  - Dense embeddings (OpenAI `text-embedding-3`, BGE, Cohere) paired with Sparse search (BM25 / SPLADE).
  - Reciprocal Rank Fusion (RRF) and Cross-Encoder Re-ranking (e.g., Cohere Rerank, BGE-Reranker).
  - Vector DBs: pgvector (PostgreSQL), Chroma, Qdrant, Pinecone, Milvus, Weaviate.
- **Query Optimization**: Query expansion, hypothetical document embeddings (HyDE), sub-query decomposition, and step-back prompting.
- **Context Injection & Generation**: Dynamic prompt assembly, citation tracking, and anti-hallucination verification.

### C. Prompt Engineering & Structured Outputs
- **Techniques**: Few-shot in-context learning, Chain-of-Thought (CoT), Tree-of-Thoughts (ToT), Directional Stimulus Prompting, System-Persona framing.
- **Structured Output Guarantees**: Using native JSON schema mode, Instructor, Outlines, or Pydantic validation with automated retry on schema violation.
- **Guardrails & Safety**: NeMo Guardrails, Guardrails AI, PII masking/redaction, and jailbreak/prompt injection detection.

---

## 2. Standard AI Service Project Blueprint (Python / FastAPI)

```text
ai_service/
├── app/
│   ├── api/
│   │   ├── v1/
│   │   │   ├── endpoints/
│   │   │   │   ├── chat.py         # Streaming & standard chat completions
│   │   │   │   ├── rag.py          # Query & document Q&A
│   │   │   │   └── ingest.py       # Document upload & vectorization
│   │   │   └── router.py
│   ├── core/
│   │   ├── config.py               # Pydantic Settings (API keys, model configs)
│   │   ├── logging.py              # Structured JSON logging
│   │   └── security.py             # Auth & API key validation
│   ├── agents/
│   │   ├── graph.py                # LangGraph / StateGraph workflow definition
│   │   ├── state.py                # Graph state definitions (TypedDict / Pydantic)
│   │   ├── nodes/                  # Agent step nodes (reasoner, tool_caller, reviewer)
│   │   └── tools/                  # Executable agent tools & integrations
│   ├── rag/
│   │   ├── chunking.py             # Splitting & document loaders
│   │   ├── embeddings.py           # Embedding client abstraction
│   │   ├── retriever.py            # Hybrid retriever + Re-ranking
│   │   └── vector_store.py         # Vector DB repository
│   ├── schemas/                    # Pydantic request/response models
│   ├── services/
│   │   ├── llm_provider.py         # Unified LLM provider (OpenAI, Gemini, Anthropic, LiteLLM)
│   │   ├── memory_service.py       # Redis / DB session memory management
│   │   └── cache_service.py        # Semantic cache (Redis + embeddings)
│   └── main.py                     # App factory, lifespan, CORS, middleware
├── tests/
│   ├── unit/
│   ├── integration/
│   └── eval/                       # RAG & LLM evaluation benchmarks
├── .env.example
├── pyproject.toml / requirements.txt
└── Dockerfile
```

---

## 3. Production Patterns & Code Templates

### A. Robust Streaming LLM Handler with Fallback & Retries (Python / Async)

```python
import os
import json
from typing import AsyncGenerator, Any, Dict, List
from openai import AsyncOpenAI
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception_type

class LLMService:
    def __init__(self, primary_model: str = "gpt-4o", fallback_model: str = "gpt-4o-mini"):
        self.client = AsyncOpenAI(api_key=os.getenv("OPENAI_API_KEY"))
        self.primary_model = primary_model
        self.fallback_model = fallback_model

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=2, max=10),
        reraise=True
    )
    async def stream_chat(
        self,
        messages: List[Dict[str, str]],
        temperature: float = 0.2,
        max_tokens: int = 2048,
    ) -> AsyncGenerator[str, None]:
        """Streams tokens safely with fallback handling."""
        try:
            stream = await self.client.chat.completions.create(
                model=self.primary_model,
                messages=messages,
                temperature=temperature,
                max_tokens=max_tokens,
                stream=True
            )
            async for chunk in stream:
                delta = chunk.choices[0].delta
                if delta and delta.content:
                    yield delta.content
        except Exception as err:
            # Fallback to secondary model if primary fails repeatedly
            fallback_stream = await self.client.chat.completions.create(
                model=self.fallback_model,
                messages=messages,
                temperature=temperature,
                max_tokens=max_tokens,
                stream=True
            )
            async for chunk in fallback_stream:
                delta = chunk.choices[0].delta
                if delta and delta.content:
                    yield delta.content
```

### B. High-Precision Hybrid RAG Retriever (pgvector + BM25 + Cross-Encoder)

```python
from typing import List
from pydantic import BaseModel

class DocumentChunk(BaseModel):
    id: str
    content: str
    metadata: dict
    score: float = 0.0

class HybridRetriever:
    def __init__(self, vector_store, reranker_client):
        self.vector_store = vector_store
        self.reranker = reranker_client

    async def retrieve(self, query: str, top_k: int = 5, fetch_k: int = 20) -> List[DocumentChunk]:
        # 1. Fetch dense vector results
        dense_results = await self.vector_store.similarity_search(query, k=fetch_k)
        
        # 2. Fetch sparse lexical results (BM25 / Keyword)
        sparse_results = await self.vector_store.keyword_search(query, k=fetch_k)
        
        # 3. Reciprocal Rank Fusion (RRF)
        fused_candidates = self._reciprocal_rank_fusion([dense_results, sparse_results], k=60)
        
        # 4. Cross-Encoder Re-ranking
        ranked_docs = await self.reranker.rerank(
            query=query,
            documents=[doc.content for doc in fused_candidates[:fetch_k]],
            top_n=top_k
        )
        
        return [fused_candidates[res.index] for res in ranked_docs]

    def _reciprocal_rank_fusion(self, results_lists: List[List[DocumentChunk]], k: int = 60) -> List[DocumentChunk]:
        scores = {}
        doc_map = {}
        for r_list in results_lists:
            for rank, doc in enumerate(r_list):
                doc_map[doc.id] = doc
                scores[doc.id] = scores.get(doc.id, 0.0) + (1.0 / (k + rank + 1))
        
        sorted_ids = sorted(scores.keys(), key=lambda d_id: scores[d_id], reverse=True)
        return [doc_map[d_id] for d_id in sorted_ids]
```

---

## 4. LLM Evaluation & Quality Benchmarks (Evals)

Always institute continuous evaluation for production AI systems:

1. **RAG Triad Metrics (via Ragas / DeepEval)**:
   - **Context Relevance**: Is retrieved context pertinent to the user query?
   - **Groundedness / Faithfulness**: Is the response derived solely from the retrieved context (no hallucinations)?
   - **Answer Relevance**: Does the final answer directly resolve the user's intent?
2. **Deterministic & Semantic Assertions**:
   - JSON Schema / Regex compliance.
   - Semantic similarity against golden ground-truth sets.
   - Cost / Latency budgets per request ($ / 1k tokens, Time-to-First-Token [TTFT]).
3. **Continuous Observability**:
   - Trace full call graphs with OpenTelemetry / Langfuse / Arize Phoenix (prompts, raw completions, tool calls, latencies, tokens consumed).

---

## 5. Security & Safety Guidelines

- **Prompt Injection Defense**: Treat all user inputs and external retrieved context as untrusted data. Sanitize and isolate data using XML delimiter tags (e.g., `<user_data>...</user_data>`).
- **Sandboxed Tool Execution**: Never run dynamically generated code without containerized, resource-constrained isolation (e.g., gVisor, Docker, or isolated WASM runtimes).
- **Cost & Rate Control**: Enforce max token caps, rate limits per IP/user, and implement semantic response caching via Redis to eliminate redundant LLM invocations.
