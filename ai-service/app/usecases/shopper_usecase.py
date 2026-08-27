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

SYSTEM_PROMPT = """Anda adalah 'Tirenn AI Shopper', asisten belanja pintar, jujur, dan ramah untuk Tirenn Commerce.

Pedoman & Aturan Utama:
1. Membantu pelanggan menemukan produk yang tepat (rekomendasi, hadiah, perbandingan harga, spesifikasi).
2. Menjawab dengan bahasa Indonesia yang ramah, santun, dan natural.
3. GUNAKAN TOOLS yang tersedia untuk mencari produk, mengecek stok asli, dan menambahkan barang ke keranjang!
4. Jika pengguna mencari produk atau menyebutkan kriteria harga/stok (misal: 'celana jeans antara 200rb - 350rb', 'rekomendasi kopi', 'produk yang ready stok'), GUNAKAN tool `search_products`.
   - Konversi singkatan harga ke angka Rupiah penuh (contoh: 50k/50rb = 50000, 250rb = 250000, 1jt = 1000000).
   - Kosongkan `min_price` atau `max_price` jika pengguna tidak menentukan batasan.
   - Gunakan `in_stock=true` jika pengguna meminta produk yang tersedia / ready.
   - Selalu isi parameter `query` dengan kata kunci pencarian yang bersih dan relevan.
5. Jika pengguna menanyakan ketersediaan stok barang spesifik, GUNAKAN tool `check_product_stock`.
6. Jika pengguna meminta memasukkan produk ke keranjang belanja, GUNAKAN tool `add_to_cart`.
   - Jika respon tool `add_to_cart` berstatus `action: auth_required`, sampaikan dengan ramah bahwa pengguna perlu login ke akunnya terlebih dahulu agar dapat memasukkan produk ke keranjang.
7. ATURAN GROUNDING KETAT: Hanya rekomendasikan produk yang benar-benar ada dalam hasil data tool! Jangan pernah mengarang produk, harga, atau SKU yang tidak ada di katalog.
8. PENTING: JANGAN menyertakan URL gambar, link gambar, atau markdown gambar (seperti 'Gambar: ![](...)') dalam teks balasan Anda! Gambar produk sudah otomatis ditampilkan secara visual oleh kartu sistem UI.

Contoh Pola Pemanggilan Tool:
- User: "Carikan sepatu lari pria di bawah 300 ribu yang ready"
  -> search_products(query="sepatu lari pria", max_price=300000, in_stock=true, category_id=2)
- User: "Ada stok Kopi Arabika Gayo?"
  -> check_product_stock(product_name_or_query="Kopi Arabika Gayo")
- User: "Masukkan Headphone AuraPro ke keranjang"
  -> add_to_cart(product_name_or_query="Headphone AuraPro", quantity=1)"""

def normalize_price(val: Optional[float]) -> Optional[float]:
    """Smart normalizer: ignore 0 or negative values; convert shorthand thousands to full Rupiah"""
    if val is None:
        return None
    try:
        val = float(val)
    except (ValueError, TypeError):
        return None
    if val <= 0:
        return None
    if 0 < val < 1000:
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
        """Dynamically generate tool definitions with live categories from database"""
        cats = self.product_repo.get_categories_map()
        cat_desc_list = [f"{k}: {v}" for k, v in cats.items()]
        cat_desc = ", ".join(cat_desc_list) if cat_desc_list else "ID Kategori Produk"

        return [
            {
                "type": "function",
                "function": {
                    "name": "search_products",
                    "description": "Mencari produk di katalog Tirenn Commerce berdasarkan makna/kebutuhan (semantic vector & keyword search), kategori, batas harga minimal/maksimal (dalam Rupiah penuh), dan status ketersediaan stok.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "query": {
                                "type": "string",
                                "description": "Kata kunci pencarian atau deskripsi kebutuhan (contoh: 'pakaian wanita', 'kopi arabika gayo', 'celana jeans pria'). Wajib diisi."
                            },
                            "min_price": {
                                "type": "number",
                                "description": "Batas harga minimal dalam angka Rupiah penuh (contoh: 200rb diisi 200000). Kosongkan jika tidak ada batas bawah."
                            },
                            "max_price": {
                                "type": "number",
                                "description": "Batas harga maksimal dalam angka Rupiah penuh (contoh: 350rb diisi 350000). Kosongkan jika tidak ada batas atas."
                            },
                            "in_stock": {
                                "type": "boolean",
                                "description": "Filter hanya produk yang masih memiliki stok tersedia (true/false)."
                            },
                            "category_id": {
                                "type": "integer",
                                "description": f"ID kategori spesifik jika diketahui ({cat_desc})."
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
                    "description": "Mengecek sisa stok dan harga aktual produk dari database PostgreSQL secara real-time berdasarkan nama produk, kata kunci, atau SKU produk.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "product_name_or_query": {
                                "type": "string",
                                "description": "Nama produk, kata kunci, atau SKU yang ingin dicek stoknya (contoh: 'Headphone AuraPro', 'ELEC-001', 'Kopi Arabika Gayo')."
                            },
                            "product_id": {
                                "type": "integer",
                                "description": "ID angka produk jika diketahui secara pasti."
                            }
                        }
                    }
                }
            },
            {
                "type": "function",
                "function": {
                    "name": "add_to_cart",
                    "description": "Menambahkan produk terpilih ke dalam keranjang belanja pelanggan.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "product_name_or_query": {
                                "type": "string",
                                "description": "Nama produk atau SKU yang ingin dimasukkan ke keranjang (contoh: 'Headphone AuraPro', 'ELEC-001')."
                            },
                            "product_id": {
                                "type": "integer",
                                "description": "ID angka produk jika diketahui."
                            },
                            "quantity": {
                                "type": "integer",
                                "description": "Jumlah kuantitas (default: 1)."
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
        min_price = normalize_price(args.get("min_price"))
        max_price = normalize_price(args.get("max_price"))
        in_stock = args.get("in_stock")

        cats = self.product_repo.get_categories_map()
        cat_name = cats.get(cat_id, "all")

        logger.info(
            f"🔎 [TOOL_PARAMS: search_products] "
            f"query='{query}' | "
            f"category_id={cat_id} ({cat_name}) | "
            f"min_price={min_price} | "
            f"max_price={max_price} | "
            f"in_stock={in_stock}"
        )

        if not query:
            query = cat_name if cat_name != "all" else "produk rekomendasi"

        results = self.search_usecase.execute(
            query=query,
            limit=12,
            category_id=cat_id,
            score_threshold=settings.CHAT_SEARCH_SCORE_THRESHOLD,
            min_price=min_price,
            max_price=max_price,
            in_stock=in_stock
        )

        if not results:
            results = self.search_usecase.execute(
                query=query,
                limit=15,
                category_id=cat_id,
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
                "price": r.price,
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
            candidates = self.search_usecase.execute(query, limit=1, score_threshold=settings.CHAT_SEARCH_FALLBACK_THRESHOLD)
            if candidates:
                prod = self.product_repo.get_product_by_id(candidates[0].id)

        if not prod:
            return {"error": f"Produk '{query or p_id}' tidak ditemukan di katalog Tirenn Commerce."}

        return {
            "product_id": prod.id,
            "name": prod.name,
            "sku": prod.sku,
            "stock_quantity": prod.stock_quantity,
            "price": float(prod.price),
            "image_url": prod.image_url,
            "in_stock": (prod.stock_quantity > 0),
            "badge": prod.badge
        }

    def _execute_add_to_cart(self, args: Dict[str, Any], is_authenticated: bool = False) -> Dict[str, Any]:
        """Tool implementation: add_to_cart (Enforces login requirement)"""
        p_id = args.get("product_id")
        query = (args.get("product_name_or_query") or "").strip()
        qty = max(1, int(args.get("quantity", 1)))

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
            candidates = self.search_usecase.execute(query, limit=1, score_threshold=settings.CHAT_SEARCH_FALLBACK_THRESHOLD)
            if candidates:
                prod = self.product_repo.get_product_by_id(candidates[0].id)

        if not prod:
            return {"error": f"Produk '{query or p_id}' tidak ditemukan untuk dimasukkan ke keranjang."}

        # Check authentication constraint
        if not is_authenticated:
            logger.info(f"🔒 [AUTH_REQUIRED: add_to_cart] Guest user attempted to add '{prod.name}' to cart.")
            return {
                "action": "auth_required",
                "product_id": prod.id,
                "sku": prod.sku,
                "name": prod.name,
                "price": float(prod.price),
                "image_url": prod.image_url,
                "stock_quantity": prod.stock_quantity,
                "message": f"Pengguna belum login (guest). Beritahukan dengan ramah bahwa untuk memasukkan produk '{prod.name}' ke keranjang belanja, pengguna wajib login terlebih dahulu.",
                "success": False
            }

        return {
            "action": "cart_added",
            "product_id": prod.id,
            "sku": prod.sku,
            "name": prod.name,
            "quantity": qty,
            "price": float(prod.price),
            "image_url": prod.image_url,
            "stock_quantity": prod.stock_quantity,
            "success": True
        }

    def _dispatch_tool(self, name: str, args: Dict[str, Any], is_authenticated: bool = False) -> Dict[str, Any]:
        """Route tool name to corresponding implementation function"""
        logger.info(f"🛠️ [TOOL_CALL_DISPATCH] Invoking tool='{name}' with raw args: {json.dumps(args, ensure_ascii=False)}")
        start_t = time.perf_counter()

        if name == "search_products":
            res = self._execute_search_products(args)
        elif name == "check_product_stock":
            res = self._execute_check_product_stock(args)
        elif name == "add_to_cart":
            res = self._execute_add_to_cart(args, is_authenticated=is_authenticated)
        else:
            logger.warning(f"⚠️ [AI_TOOL_UNKNOWN] tool='{name}' | received_args={args}")
            res = {"error": f"Tool '{name}' tidak dikenal"}

        elapsed_ms = (time.perf_counter() - start_t) * 1000.0
        status = "error" if "error" in res else ("auth_required" if res.get("action") == "auth_required" else "success")
        logger.info(
            f"✅ [TOOL_EXECUTION_COMPLETE] tool='{name}' | status='{status}' | "
            f"params={json.dumps(args, ensure_ascii=False)} | latency={elapsed_ms:.1f}ms"
        )
        return res

    async def chat(
        self,
        messages: List[ChatMessage],
        is_authenticated: bool = False,
        user_name: Optional[str] = None
    ) -> ChatShopperResult:
        """Main conversational shopper loop with Ollama, Tool Dispatching, and Logging"""
        start_time = time.perf_counter()
        last_user_msg = messages[-1].content if messages else ""

        formatted_messages = [{"role": "system", "content": SYSTEM_PROMPT}] + [
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

                    # Execute tool via clean dispatcher with authentication state
                    tool_result = self._dispatch_tool(tool_name, tool_args, is_authenticated=is_authenticated)

                    if "_full_products" in tool_result:
                        suggested_products.extend(tool_result.pop("_full_products"))
                    if tool_result.get("action") in ("cart_added", "auth_required"):
                        cart_action = tool_result

                    executed_tools_data.append({
                        "name": tool_name,
                        "args": tool_args,
                        "output": tool_result
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
