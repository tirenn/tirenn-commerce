import logging
from typing import List, Optional

from app.domain.chat import ChatMessage, ChatShopperResult
from app.repositories.llm_repository import LLMRepository
from app.repositories.product_repository import ProductRepository
from app.usecases.search_usecase import SearchUseCase
from app.harness.agent import AgentHarness
from app.harness.tools.catalog_tools import SearchProductsTool, CheckProductStockTool
from app.harness.tools.cart_tools import AddToCartTool

logger = logging.getLogger("ai-service.usecase.shopper")

SYSTEM_PROMPT = """You are 'Tirenn AI Shopper', a smart, honest, friendly, and bilingual AI shopping assistant for Tirenn Commerce.

Bilingual & Communication Guidelines:
1. Automatically detect the user's language (Bahasa Indonesia or English) and reply fluently in the SAME language.
2. Be polite, concise, and helpful with product recommendations, comparisons, and purchasing advice.
3. MANDATORY TOOL CALLING:
   - If the user searches for products, asks for recommendations, or mentions product types (e.g. 'cari celana panjang pria', 'find wireless headphones', 'recommend specialty coffee', 'tas selempang wanita'), you MUST call the `search_products` tool.
   - If the user asks for remaining stock or price of a specific product, call `check_product_stock`.
   - If the user wants to add an item to their shopping cart (e.g. 'masukkan ke keranjang', 'add to cart', 'buy this'), call `add_to_cart`.
4. Parameters:
   - Convert shorthand price amounts to full numbers (e.g. 50k/50rb = 50000, 250rb = 250000, $20 = 320000 IDR, 1jt = 1000000).
   - Leave `min_price` or `max_price` empty if user does not specify a budget.
   - Set `in_stock=true` if user asks for ready/available stock.
5. STRICT GROUNDING & RELEVANCE RULE: Only recommend and talk about products verified by the tools! Never hallucinate products, prices, or SKUs not in the catalog.
6. NO IMAGE MARKDOWN: Do NOT include image URLs or markdown `![](...)` in your text reply; the system UI automatically displays product cards visually.

Contoh Pola / Examples:
- User: "Find running shoes for men under 30 dollars" -> search_products(query="running shoes", max_price=500000, in_stock=true, category_id=2)
- User: "Carikan celana panjang pria yang ready" -> search_products(query="celana panjang pria", in_stock=true, category_id=2)
- User: "Add AuraSound headphones to cart" -> add_to_cart(product_name_or_query="AuraSound", quantity=1)
"""

class ShopperUseCase:
    """Enterprise Shopper UseCase powered by Tirenn Agent Harness"""

    def __init__(
        self,
        llm_repo: LLMRepository,
        product_repo: ProductRepository,
        search_usecase: SearchUseCase
    ):
        self.llm_repo = llm_repo
        self.product_repo = product_repo
        self.search_usecase = search_usecase

        # Register tools in harness
        self.search_tool = SearchProductsTool(product_repo, search_usecase)
        self.stock_tool = CheckProductStockTool(product_repo, search_usecase)
        self.cart_tool = AddToCartTool(product_repo, search_usecase)

        self.tools = [self.search_tool, self.stock_tool, self.cart_tool]

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
        user_name: Optional[str] = None
    ) -> ChatShopperResult:
        """Delegate conversational shopping turn to the Agent Harness"""
        context = {
            "is_authenticated": is_authenticated,
            "user_name": user_name
        }
        return await self.harness.run(messages=messages, context=context)
