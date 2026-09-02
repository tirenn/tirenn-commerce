import logging
from typing import List, Optional, Dict, Any

from app.core.config import settings
from app.core.llm_cache import LLMSemanticCache
from app.core.prompt_loader import load_prompt
from app.domain.chat import ChatMessage, ChatShopperResult
from app.repositories.llm_repository import LLMRepository
from app.repositories.product_repository import ProductRepository
from app.repositories.embedding_repository import EmbeddingRepository
from app.usecases.search_usecase import SearchUseCase
from app.usecases.knowledge_usecase import KnowledgeUseCase
from app.harness.agent import AgentHarness
from app.harness.tools.customer import (
    SearchProductsTool,
    GetProductStockTool,
    GetProductDetailTool,
    SearchStorePoliciesAndSOPTool,
    AddToCartTool,
    ViewCartTool,
)
from app.repositories.session_repository import SessionRepository

logger = logging.getLogger("ai-service.usecase.shopper")


class ShopperUseCase:
    """Enterprise Shopper UseCase powered by Tirenn Agent Harness, Vector RAG & Redis Session Store"""

    def __init__(
        self,
        llm_repo: LLMRepository,
        product_repo: ProductRepository,
        search_usecase: SearchUseCase,
        knowledge_usecase: Optional[KnowledgeUseCase] = None,
        session_repo: Optional[SessionRepository] = None,
        embedding_repo: Optional[EmbeddingRepository] = None,
        llm_cache: Optional[LLMSemanticCache] = None,
        system_prompt: Optional[str] = None
    ):
        self.llm_repo = llm_repo
        self.product_repo = product_repo
        self.search_usecase = search_usecase
        self.knowledge_usecase = knowledge_usecase
        self.session_repo = session_repo or SessionRepository()
        self.embedding_repo = embedding_repo
        self.llm_cache = llm_cache or LLMSemanticCache()
        self.system_prompt = system_prompt or load_prompt("shopper_agent.md")

        # Register tools in harness
        self.search_tool = SearchProductsTool(product_repo, search_usecase)
        self.detail_tool = GetProductDetailTool(product_repo)
        self.stock_tool = GetProductStockTool(product_repo)
        self.cart_tool = AddToCartTool(product_repo)
        self.view_cart_tool = ViewCartTool(product_repo)
        self.knowledge_tool = SearchStorePoliciesAndSOPTool(knowledge_usecase)

        self.tools = [
            self.search_tool,
            self.detail_tool,
            self.stock_tool,
            self.cart_tool,
            self.view_cart_tool,
            self.knowledge_tool,
        ]

        # Initialize Agent Harness
        self.harness = AgentHarness(
            llm_repo=self.llm_repo,
            tools=self.tools,
            system_prompt=self.system_prompt,
            max_iterations=settings.MAX_AGENT_ITERATIONS
        )

    async def chat(
        self,
        messages: List[ChatMessage],
        is_authenticated: bool = False,
        user_name: Optional[str] = None,
        session_id: Optional[str] = None,
        cart_items: Optional[List[Dict[str, Any]]] = None
    ) -> ChatShopperResult:
        """Delegate conversational shopping and SOP inquiries to the Agent Harness with 2-Tier LLM Semantic Response Cache"""
        import time
        start_time = time.perf_counter()
        last_user_msg = messages[-1] if messages else ChatMessage(role="user", content="")
        query_text = (last_user_msg.content or "").strip()

        # ----------------------------------------------------------------------
        # 1. Check 2-Tier LLM Semantic Response Cache (< 5ms)
        # ----------------------------------------------------------------------
        query_vec = None
        if query_text and self.embedding_repo and settings.LLM_CACHE_ENABLED:
            try:
                query_vec = self.embedding_repo.encode(query_text)
            except Exception as e:
                logger.warning(f"Failed to encode query vector for LLM Cache: {e}")

        # Only check cache for general inquiry turns (do not bypass live cart item actions)
        is_cart_mutation = any(
            w in query_text.lower()
            for w in ["masukkan ke keranjang", "tambah ke keranjang", "add to cart", "masukin cart"]
        )

        if not is_cart_mutation and query_text and settings.LLM_CACHE_ENABLED:
            cached_turn = self.llm_cache.get_cached_turn(
                query=query_text,
                query_vector=query_vec,
                scope="shopper"
            )
            if cached_turn:
                elapsed_ms = (time.perf_counter() - start_time) * 1000.0
                logger.info(f"⚡ [LLM_CACHE_RETURN] Fast response delivered in {elapsed_ms:.2f}ms for '{query_text[:50]}'")

                cached_res = ChatShopperResult(
                    reply=cached_turn.get("reply", ""),
                    tool_calls=cached_turn.get("tool_calls", []),
                    suggested_products=cached_turn.get("suggested_products", []),
                    cart_action=cached_turn.get("cart_action"),
                    latency_ms=elapsed_ms
                )

                # Append turn to session history in Redis
                if session_id and self.session_repo and query_text:
                    new_turn = [
                        last_user_msg,
                        ChatMessage(role="assistant", content=cached_res.reply)
                    ]
                    self.session_repo.append_messages(session_id, new_turn)

                return cached_res

        # ----------------------------------------------------------------------
        # 2. Cache Miss: Execute Full Agent ReAct Harness Turn
        # ----------------------------------------------------------------------
        # Retrieve bounded sliding window history from Redis List (default last 10 messages)
        if session_id and self.session_repo:
            stored_history = self.session_repo.get_history(session_id, limit=settings.SESSION_HISTORY_LIMIT)
            if stored_history:
                effective_messages = stored_history + [last_user_msg]
            else:
                effective_messages = messages[-settings.SESSION_HISTORY_LIMIT:]
        else:
            effective_messages = messages[-settings.SESSION_HISTORY_LIMIT:]

        # Dynamically retrieve live taxonomy from PostgreSQL database
        live_taxonomy = self.product_repo.get_taxonomy_prompt_text()

        context = {
            "is_authenticated": is_authenticated,
            "user_name": user_name,
            "session_id": session_id,
            "cart_items": cart_items or [],
            "taxonomy": live_taxonomy
        }

        result = await self.harness.run(messages=effective_messages, context=context)

        # ----------------------------------------------------------------------
        # 3. Store result in 2-Tier LLM Semantic Response Cache
        # ----------------------------------------------------------------------
        if not is_cart_mutation and query_text and query_vec and result.reply and settings.LLM_CACHE_ENABLED:
            cache_payload = {
                "reply": result.reply,
                "tool_calls": result.tool_calls,
                "suggested_products": result.suggested_products,
                "cart_action": result.cart_action
            }
            self.llm_cache.set_cached_turn(
                query=query_text,
                query_vector=query_vec,
                payload=cache_payload,
                scope="shopper"
            )

        # Atomically append new exchange (user prompt + assistant reply) to Redis List with auto-trim & TTL
        if session_id and self.session_repo and query_text:
            new_turn = [
                last_user_msg,
                ChatMessage(role="assistant", content=result.reply)
            ]
            self.session_repo.append_messages(session_id, new_turn)

        return result

    def delete_session(self, session_id: str) -> bool:
        """Delete session history from Redis when session ends or is reset"""
        if self.session_repo and session_id:
            return self.session_repo.delete_session(session_id)
        return False

