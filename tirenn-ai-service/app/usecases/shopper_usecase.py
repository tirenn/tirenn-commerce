import logging
from typing import List, Optional, Dict, Any

from app.domain.chat import ChatMessage, ChatShopperResult
from app.repositories.llm_repository import LLMRepository
from app.repositories.product_repository import ProductRepository
from app.usecases.search_usecase import SearchUseCase
from app.usecases.knowledge_usecase import KnowledgeUseCase
from app.harness.agent import AgentHarness
from app.harness.tools.catalog_tools import SearchProductsTool, GetProductStockTool, GetProductDetailTool, SearchStorePoliciesAndSOPTool
from app.repositories.session_repository import SessionRepository
from app.harness.tools.cart_tools import AddToCartTool, ViewCartTool

logger = logging.getLogger("ai-service.usecase.shopper")

SYSTEM_PROMPT = """You are 'Tirenn AI Shopper', a smart, honest, friendly, and bilingual AI shopping assistant for Tirenn Commerce.

STRICT BILINGUAL LANGUAGE POLICY:
1. LANGUAGE DETECTION & MATCHING:
   - Carefully inspect the language of the user's latest message.
   - If the user writes in ENGLISH (e.g. 'find shoes', 'tell me about this product', 'check stock', 'add to cart', 'recommend women dress') -> YOU MUST RESPOND 100% IN ENGLISH.
   - If the user writes in BAHASA INDONESIA (e.g. 'cari sepatu', 'jelaskan produk ini', 'cek stok', 'masukkan ke keranjang') -> YOU MUST RESPOND 100% IN BAHASA INDONESIA.
   - NEVER reply in Indonesian if the user asks in English, and NEVER reply in English if the user asks in Indonesian.

2. AVAILABLE TOOLS & ACTIONS:
   - 1. `search_products(query)`: Call when user searches or asks for product recommendations (e.g. 'cari celana panjang pria', 'recommend wireless headphones').
   - 2. `get_product_detail(sku)`: Call when user asks for full specifications, materials, or features of a specific product by SKU.
   - 3. `get_product_stock(sku)`: Call when user asks about remaining inventory, stock status, or price by SKU.
   - 4. `add_to_cart(sku, qty)`: Call when user wants to add an item to the shopping cart by SKU and quantity (default 1).
   - 5. `view_cart()`: Call when user asks to view what is currently inside their shopping cart.
   - 6. `search_store_policies_and_sop(query)`: Call when user asks about return/warranty policies, payment methods, delivery times, or shopping procedures.

3. STRICT GROUNDING RULE: Only provide facts, prices, policies, and products verified by the tools. Never hallucinate.
4. RECOMMENDATION LIMIT: Present at most 6 products from the verified tool results. Never list more than 6 products.
5. NO IMAGE MARKDOWN: Do NOT include image URLs or markdown `![](...)` in your text reply.
"""

class ShopperUseCase:
    """Enterprise Shopper UseCase powered by Tirenn Agent Harness, Vector RAG & Redis Session Store"""

    def __init__(
        self,
        llm_repo: LLMRepository,
        product_repo: ProductRepository,
        search_usecase: SearchUseCase,
        knowledge_usecase: Optional[KnowledgeUseCase] = None,
        session_repo: Optional[SessionRepository] = None
    ):
        self.llm_repo = llm_repo
        self.product_repo = product_repo
        self.search_usecase = search_usecase
        self.knowledge_usecase = knowledge_usecase
        self.session_repo = session_repo or SessionRepository()

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
            system_prompt=SYSTEM_PROMPT,
            max_iterations=5
        )

    async def chat(
        self,
        messages: List[ChatMessage],
        is_authenticated: bool = False,
        user_name: Optional[str] = None,
        session_id: Optional[str] = None,
        cart_items: Optional[List[Dict[str, Any]]] = None
    ) -> ChatShopperResult:
        """Delegate conversational shopping and SOP inquiries to the Agent Harness with Redis session support"""
        effective_messages = list(messages)

        # If session_id provided, merge with Redis persistent history
        if session_id and self.session_repo:
            stored_history = self.session_repo.get_history(session_id)
            if stored_history:
                if len(messages) == 1:
                    effective_messages = stored_history + [messages[0]]
                elif len(messages) > len(stored_history):
                    effective_messages = messages

        context = {
            "is_authenticated": is_authenticated,
            "user_name": user_name,
            "session_id": session_id,
            "cart_items": cart_items or []
        }

        result = await self.harness.run(messages=effective_messages, context=context)

        # Persist full updated history into Redis with auto-expiring TTL
        if session_id and self.session_repo:
            full_history = effective_messages + [ChatMessage(role="assistant", content=result.reply)]
            self.session_repo.save_history(session_id, full_history)

        return result

    def delete_session(self, session_id: str) -> bool:
        """Delete session history from Redis when session ends or is reset"""
        if self.session_repo and session_id:
            return self.session_repo.delete_session(session_id)
        return False
