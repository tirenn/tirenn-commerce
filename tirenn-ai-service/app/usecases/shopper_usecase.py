import time
import json
import logging
from typing import List, Dict, Any, Optional

from app.core.config import settings
from app.domain.chat import ChatMessage, ChatShopperResult
from app.repositories.llm_repository import LLMRepository
from app.repositories.product_repository import ProductRepository
from app.usecases.search_usecase import SearchUseCase

logger = logging.getLogger("ai-service.usecase.shopper")

SYSTEM_PROMPT = """You are 'Tirenn AI Shopper', a smart, honest, friendly, and bilingual AI shopping assistant for Tirenn Commerce.

Bilingual & Communication Guidelines:
1. Automatically detect the user's language (Bahasa Indonesia or English) and reply fluently in the SAME language.
2. Be polite, concise, and helpful with product recommendations, comparisons, and purchasing advice.
3. MANDATORY TOOL CALLING:
   - If the user searches for products, asks for recommendations, or mentions product types (e.g. 'cari tas selempang', 'find men running shoes', 'recommend coffee beans', 'headphones under $50'), you MUST call the `search_products` tool.
   - If the user asks for remaining stock or price of a specific product, call `check_product_stock`.
   - If the user wants to add an item to their shopping cart (e.g. 'masukkan ke keranjang', 'add to cart', 'buy this'), call `add_to_cart`.
4. Parameters:
   - Convert shorthand price amounts to full numbers (e.g. 50k/50rb = 50000, 250rb = 250000, $20 = 320000 IDR, 1jt = 1000000).
   - Leave `min_price` or `max_price` empty if user does not specify a budget.
   - Set `in_stock=true` if user asks for ready/available stock.
5. STRICT GROUNDING RULE: Only recommend and talk about products returned by the tools! Never hallucinate products, prices, or SKUs not in the catalog.
6. NO IMAGE MARKDOWN: Do NOT include image URLs or markdown `![](...)` in your text reply; the system UI automatically displays product cards visually.

Contoh Pola / Examples:
- User: "Find running shoes for men under 30 dollars" -> search_products(query="running shoes", max_price=500000, in_stock=true, category_id=2)
- User: "Carikan kemeja flanel pria yang ready" -> search_products(query="kemeja flanel", in_stock=true, category_id=2)
- User: "Add AuraPro headphones to cart" -> add_to_cart(product_name_or_query="AuraPro", quantity=1)
"""

def normalize_price(val: Optional[float]) -> Optional[float]:
    """Smart normalizer: convert dollars or shorthand thousands to full Rupiah"""
    if val is None:
        return None
    try:
        val = float(val)
    except (ValueError, TypeError):
        return None
    if val <= 0:
        return None
    # If amount is very small (likely USD e.g. $10 - $200), convert to approx IDR (1 USD = 16,000 IDR)
    if val < 500:
        return val * 16000.0
    if 500 <= val < 1000:
        return val * 1000.0
    return val

class ShopperUseCase:
    """UseCase orchestrating conversational agentic shopper loop, dynamic tool generation, and grounding"""

    def __init__(
        self,
        llm_repo: LLMRepository,
        product_repo: ProductRepository,
        search_usecase: SearchUseCase
    ):
        self.llm_repo = llm_repo
        self.product_repo = product_repo
        self.search_usecase = search_usecase

    def _build_tools_schema(self) -> List[Dict[str, Any]]:
        """Dynamically generate tool definitions with live categories and subcategories from database"""
        cats = self.product_repo.get_categories_map()
        cat_desc_list = [f"{k}: {v}" for k, v in cats.items()]
        cat_desc = ", ".join(cat_desc_list) if cat_desc_list else "1: Electronics, 2: Men Fashion, 3: Women Fashion, 4: Food & Drink, 5: Beauty"

        sub_cats = self.product_repo.get_sub_categories_map()
        sub_desc_list = [f"{k}: {v}" for k, v in sub_cats.items()]
        sub_desc = ", ".join(sub_desc_list) if sub_desc_list else "ID Sub-Category"

        return [
            {
                "type": "function",
                "function": {
                    "name": "search_products",
                    "description": "Search product catalog using semantic similarity, keyword matching, category, and price filters.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "query": {
                                "type": "string",
                                "description": "Search query or product description (e.g. 'wireless headphones', 'kemeja flanel', 'tas wanita', 'running shoes'). Required."
                            },
                            "min_price": {
                                "type": "number",
                                "description": "Minimum price in IDR or USD. Empty if no lower bound."
                            },
                            "max_price": {
                                "type": "number",
                                "description": "Maximum price in IDR or USD. Empty if no upper bound."
                            },
                            "in_stock": {
                                "type": "boolean",
                                "description": "Filter only products with available stock (true/false)."
                            },
                            "category_id": {
                                "type": "integer",
                                "description": f"Main Category ID ({cat_desc})."
                            },
                            "sub_category_id": {
                                "type": "integer",
                                "description": f"Sub-Category ID ({sub_desc})."
                            }
                        },
                        "required": ["query"]
                    }
                }
            },
            {
                "type": "function",
                "function": {
                    "name": "check_product_stock",
                    "description": "Check real-time stock quantity and price of a product by name, keyword, or SKU.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "product_name_or_query": {
                                "type": "string",
                                "description": "Product name or SKU to check (e.g. 'AuraPro', 'AUD-001', 'Kopi Gayo')."
                            },
                            "product_id": {
                                "type": "integer",
                                "description": "Product ID if known."
                            }
                        }
                    }
                }
            },
            {
                "type": "function",
                "function": {
                    "name": "add_to_cart",
                    "description": "Add a selected product to customer's shopping cart (works for both guests and logged-in users).",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "product_name_or_query": {
                                "type": "string",
                                "description": "Product name or SKU to add to cart (e.g. 'AuraPro', 'AUD-001')."
                            },
                            "product_id": {
                                "type": "integer",
                                "description": "Product ID if known."
                            },
                            "quantity": {
                                "type": "integer",
                                "description": "Quantity count (default: 1)."
                            }
                        }
                    }
                }
            }
        ]

    def _execute_search_products(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Tool implementation: search_products"""
        query = (args.get("query") or "").strip()
        cat_id = int(args.get("category_id") or 0)
        sub_cat_id = int(args.get("sub_category_id") or 0)
        min_price = normalize_price(args.get("min_price"))
        max_price = normalize_price(args.get("max_price"))
        in_stock = args.get("in_stock")

        cats = self.product_repo.get_categories_map()
        cat_name = cats.get(cat_id, "all")

        logger.info(
            f"🔎 [TOOL_PARAMS: search_products] "
            f"query='{query}' | "
            f"category_id={cat_id} ({cat_name}) | "
            f"sub_category_id={sub_cat_id} | "
            f"min_price={min_price} | "
            f"max_price={max_price} | "
            f"in_stock={in_stock}"
        )

        results = self.search_usecase.execute(
            query=query,
            limit=settings.CHAT_SEARCH_LIMIT,
            category_id=cat_id,
            sub_category_id=sub_cat_id,
            score_threshold=settings.CHAT_SEARCH_SCORE_THRESHOLD,
            min_price=min_price,
            max_price=max_price,
            in_stock=in_stock
        )

        # Multi-stage adaptive fallback for broad queries
        if not results:
            results = self.search_usecase.execute(
                query=query,
                limit=settings.CHAT_SEARCH_LIMIT,
                category_id=cat_id,
                sub_category_id=sub_cat_id,
                score_threshold=settings.CHAT_SEARCH_FALLBACK_THRESHOLD,
                min_price=min_price,
                max_price=max_price,
                in_stock=in_stock
            )

        if not results and cat_id > 0:
            results = self.search_usecase.execute(
                query=cat_name if cat_name != "all" else query,
                limit=15,
                category_id=cat_id,
                sub_category_id=sub_cat_id,
                score_threshold=0.0,
                min_price=min_price,
                max_price=max_price,
                in_stock=in_stock
            )

        formatted = [
            {
                "id": r.id,
                "name": r.name,
                "sku": r.sku,
                "category_id": r.category_id,
                "sub_category_id": r.sub_category_id,
                "sub_category_name": r.sub_category_name,
                "price": r.price,
                "currency": r.currency,
                "image_url": r.image_url,
                "stock_quantity": r.stock_quantity,
                "in_stock": r.stock_quantity > 0,
                "score": round(r.score, 3)
            }
            for r in results
        ]

        llm_products = [
            {
                "id": p["id"],
                "name": p["name"],
                "sku": p["sku"],
                "price": p["price"],
                "currency": p["currency"],
                "stock_quantity": p["stock_quantity"],
                "in_stock": p["in_stock"]
            }
            for p in formatted
        ]

        return {"found_count": len(formatted), "products": llm_products, "_full_products": formatted}

    def _execute_check_product_stock(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Tool implementation: check_product_stock"""
        p_id = args.get("product_id")
        query = (args.get("product_name_or_query") or "").strip()

        logger.info(
            f"📦 [TOOL_PARAMS: check_product_stock] "
            f"product_name_or_query='{query}' | "
            f"product_id={p_id}"
        )

        prod = None
        if p_id and int(p_id) <= 1000:
            prod = self.product_repo.get_product_by_id(int(p_id))

        if not prod and query:
            prod = self.product_repo.get_product_by_sku_or_name(query)

        if not prod and query:
            search_res = self.search_usecase.execute(
                query=query,
                limit=1,
                score_threshold=0.10
            )
            if search_res:
                prod = self.product_repo.get_product_by_id(search_res[0].id)

        if not prod:
            return {"status": "not_found", "message": f"Product '{query or p_id}' not found in catalog."}

        return {
            "status": "found",
            "id": prod.id,
            "name": prod.name,
            "sku": prod.sku,
            "price": prod.price,
            "currency": prod.currency,
            "stock_quantity": prod.stock_quantity,
            "in_stock": prod.stock_quantity > 0,
            "_full_product": {
                "id": prod.id,
                "name": prod.name,
                "sku": prod.sku,
                "price": prod.price,
                "currency": prod.currency,
                "image_url": prod.image_url,
                "stock_quantity": prod.stock_quantity,
                "in_stock": prod.stock_quantity > 0,
                "score": 1.0
            }
        }

    def _execute_add_to_cart(self, args: Dict[str, Any], is_authenticated: bool = True) -> Dict[str, Any]:
        """Tool implementation: add_to_cart (available for all users including guests)"""
        p_id = args.get("product_id")
        query = (args.get("product_name_or_query") or "").strip()
        qty = int(args.get("quantity") or 1)
        if qty <= 0:
            qty = 1

        logger.info(
            f"🛒 [TOOL_PARAMS: add_to_cart] "
            f"product_name_or_query='{query}' | "
            f"product_id={p_id} | "
            f"quantity={qty} | "
            f"is_authenticated={is_authenticated}"
        )

        prod = None
        if p_id and int(p_id) <= 1000:
            prod = self.product_repo.get_product_by_id(int(p_id))

        if not prod and query:
            prod = self.product_repo.get_product_by_sku_or_name(query)

        if not prod and query:
            search_res = self.search_usecase.execute(
                query=query,
                limit=1,
                score_threshold=0.10
            )
            if search_res:
                prod = self.product_repo.get_product_by_id(search_res[0].id)

        if not prod:
            return {
                "action": "not_found",
                "message": f"Cannot add to cart: Product '{query or p_id}' not found."
            }

        if prod.stock_quantity <= 0:
            return {
                "action": "out_of_stock",
                "message": f"Sorry, '{prod.name}' is currently out of stock."
            }

        actual_qty = min(qty, prod.stock_quantity)

        return {
            "action": "cart_added",
            "message": f"'{prod.name}' ({actual_qty} pcs) has been successfully added to your shopping cart.",
            "product": {
                "id": prod.id,
                "name": prod.name,
                "sku": prod.sku,
                "price": prod.price,
                "currency": prod.currency,
                "image_url": prod.image_url,
                "stock_quantity": prod.stock_quantity,
                "quantity": actual_qty
            },
            "_full_product": {
                "id": prod.id,
                "name": prod.name,
                "sku": prod.sku,
                "price": prod.price,
                "currency": prod.currency,
                "image_url": prod.image_url,
                "stock_quantity": prod.stock_quantity,
                "in_stock": True,
                "score": 1.0
            }
        }

    async def chat(
        self,
        messages: List[ChatMessage],
        is_authenticated: bool = False,
        user_name: Optional[str] = None
    ) -> ChatShopperResult:
        """Execute full conversational agentic shopping turn with tools and smart fallbacks"""
        start_time = time.perf_counter()
        last_user_msg = messages[-1].content if messages else ""

        # Personalized user header
        system_text = SYSTEM_PROMPT
        if user_name:
            system_text += f"\nCustomer context: User name is '{user_name}' (authenticated)."

        formatted_messages = [{"role": "system", "content": system_text}] + [
            {"role": m.role, "content": m.content} for m in messages
        ]

        executed_tools_data: List[Dict[str, Any]] = []
        suggested_products: List[Dict[str, Any]] = []
        cart_action: Optional[Dict[str, Any]] = None
        final_reply = ""

        try:
            tools = self._build_tools_schema()

            # 1. Turn 1: Send conversation with tools to Ollama
            assistant_msg = await self.llm_repo.chat(
                messages=formatted_messages,
                tools=tools,
                temperature=0.0
            )

            # 2. Check for Tool Calls
            tool_calls = assistant_msg.get("tool_calls", [])

            if tool_calls:
                formatted_messages.append(assistant_msg)

                for tc in tool_calls:
                    func = tc.get("function", {})
                    tool_name = func.get("name")
                    raw_args = func.get("arguments", {})

                    if isinstance(raw_args, str):
                        try:
                            tool_args = json.loads(raw_args)
                        except Exception:
                            tool_args = {}
                    else:
                        tool_args = raw_args

                    # Execute tool via clean dispatcher
                    tool_start = time.perf_counter()
                    if tool_name == "search_products":
                        tool_result = self._execute_search_products(tool_args)
                        full_prods = tool_result.pop("_full_products", [])
                        suggested_products.extend(full_prods)

                    elif tool_name == "check_product_stock":
                        tool_result = self._execute_check_product_stock(tool_args)
                        full_prod = tool_result.pop("_full_product", None)
                        if full_prod:
                            suggested_products.append(full_prod)

                    elif tool_name == "add_to_cart":
                        tool_result = self._execute_add_to_cart(tool_args, is_authenticated=is_authenticated)
                        full_prod = tool_result.pop("_full_product", None)
                        if full_prod:
                            suggested_products.append(full_prod)
                        cart_action = tool_result
                    else:
                        tool_result = {"error": f"Unknown tool '{tool_name}'"}

                    tool_dur_ms = (time.perf_counter() - tool_start) * 1000.0
                    logger.info(f"✅ [TOOL_EXECUTION_COMPLETE] tool='{tool_name}' | status='success' | latency={tool_dur_ms:.1f}ms")

                    executed_tools_data.append({
                        "name": tool_name,
                        "params": tool_args,
                        "status": "success",
                        "result": tool_result
                    })

                    formatted_messages.append({
                        "role": "tool",
                        "content": json.dumps(tool_result, ensure_ascii=False)
                    })

                # 3. Turn 2: Synthesize grounded response
                followup_msg = await self.llm_repo.chat(
                    messages=formatted_messages,
                    temperature=0.3
                )
                final_reply = followup_msg.get("content", "")
            else:
                final_reply = assistant_msg.get("content", "")
                # Bilingual intent fallback for compact models: If user has shopping/search intent, auto-retrieve products
                search_triggers = [
                    "cari", "rekomendasi", "produk", "barang", "harga", "stok", "tas", "baju", "sepatu",
                    "celana", "kopi", "kaos", "hoodie", "dompet", "jaket", "sandal", "pakaian",
                    "elektronik", "headphone", "jam", "dress", "rok", "blouse", "beli", "ada", "mau", "diskon",
                    "find", "search", "recommend", "show", "product", "shoes", "shirt", "pants", "bag",
                    "coffee", "tea", "skincare", "perfume", "price", "stock", "cheap", "best", "buy"
                ]
                if any(w in last_user_msg.lower() for w in search_triggers):
                    logger.info(f"🔎 Smart Bilingual Intent Fallback: Auto-retrieving products for query='{last_user_msg}'")
                    fallback_res = self._execute_search_products(args={"query": last_user_msg})
                    full_prods = fallback_res.get("_full_products", [])
                    if full_prods:
                        suggested_products = full_prods
                        executed_tools_data.append({
                            "name": "search_products",
                            "params": {"query": last_user_msg},
                            "status": "success",
                            "result": fallback_res
                        })
                        if not final_reply or len(final_reply.strip()) < 10:
                            is_english = any(w in last_user_msg.lower() for w in ["find", "search", "recommend", "show", "product", "shoes", "shirt", "bag", "what", "how", "buy"])
                            if is_english:
                                final_reply = f"Here are the best matching products for '{last_user_msg}' available at Tirenn Commerce:"
                            else:
                                final_reply = f"Berikut adalah rekomendasi produk terbaik untuk '{last_user_msg}' yang tersedia di Tirenn Commerce:"

                # Universal Fallback: If no products retrieved yet and user query has substance, perform semantic search
                if not suggested_products and len(last_user_msg.strip()) >= 2:
                    logger.info(f"🔎 Universal Product Search Fallback for query='{last_user_msg}'")
                    fallback_res = self._execute_search_products(args={"query": last_user_msg})
                    full_prods = fallback_res.get("_full_products", [])
                    if full_prods:
                        suggested_products = full_prods
                        executed_tools_data.append({
                            "name": "search_products",
                            "params": {"query": last_user_msg},
                            "status": "success",
                            "result": fallback_res
                        })
                        if not final_reply or len(final_reply.strip()) < 10:
                            is_english = any(w in last_user_msg.lower() for w in ["find", "search", "recommend", "show", "product", "shoes", "shirt", "bag", "what", "how", "buy"])
                            if is_english:
                                final_reply = f"Here are the products found in our catalog for '{last_user_msg}':"
                            else:
                                final_reply = f"Berikut produk yang ditemukan di katalog untuk '{last_user_msg}':"

        except Exception as e:
            logger.error(f"ShopperUseCase error: {e}", exc_info=True)
            final_reply = "Maaf, terjadi kendala saat memproses rekomendasi belanja Anda. Silakan coba kembali sesaat lagi."

        elapsed_ms = (time.perf_counter() - start_time) * 1000.0
        called_tool_names = [t["name"] for t in executed_tools_data]

        logger.info(
            f"AI_RESPONSE_LOG: user_query='{last_user_msg}' | "
            f"tools_called={len(executed_tools_data)} | "
            f"tool_names={called_tool_names} | "
            f"products_suggested={len(suggested_products)} | "
            f"is_authenticated={is_authenticated} | "
            f"latency={elapsed_ms:.1f}ms"
        )

        return ChatShopperResult(
            reply=final_reply,
            tool_calls=executed_tools_data,
            suggested_products=suggested_products,
            cart_action=cart_action,
            latency_ms=elapsed_ms
        )
