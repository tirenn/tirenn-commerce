import time
import json
import logging
from typing import List, Dict, Any, Optional
import httpx
from app.core.config import settings
from app.services.vector_search import vector_service

logger = logging.getLogger("ai-service.shopper")

SYSTEM_PROMPT = """Anda adalah 'Tirenn AI Shopper', asisten belanja pintar dan ramah untuk Tirenn Commerce.
Tugas Anda:
1. Membantu pelanggan menemukan produk yang tepat (rekomendasi, hadiah, perbandingan harga, spesifikasi).
2. Menjawab dengan bahasa Indonesia yang ramah, santun, dan natural.
3. Gunakan Tools yang tersedia untuk mencari produk, mengecek stok asli, dan menambahkan barang ke keranjang!
4. Jika pengguna mencari produk atau menyebutkan kriteria harga/stok (misal: 'celana jeans antara 200rb - 350rb', 'rekomendasi kopi', 'produk yang ready stok'), GUNAKAN tool `search_products`.
   - Jika pengguna tidak menyebutkan batas harga minimal/maksimal, kosongkan atau jangan isi parameter `min_price` dan `max_price`.
   - Konversi singkatan harga ke angka Rupiah penuh (contoh: 200rb = 200000, 350rb = 350000).
   - Gunakan `in_stock=true` jika pengguna hanya ingin produk yang ready/tersedia stoknya.
   - PENTING: Selalu isi parameter `query` dengan kata kunci pencarian yang relevan (jangan pernah mengosongkan query).
5. Jika pengguna menanyakan ketersediaan stok barang, GUNAKAN tool `check_product_stock` dan masukkan nama produknya di `product_name_or_query`.
6. Jangan mengarang harga atau SKU. Selalu gunakan data riil dari tools.
7. PENTING: JANGAN menyertakan URL gambar, link gambar, atau markdown gambar (seperti 'Gambar: ![](...)') dalam teks balasan Anda! Gambar produk sudah otomatis ditampilkan secara visual oleh kartu sistem UI."""

# Tool definitions in standard JSON Schema format for Ollama / Qwen 2.5
SHOPPER_TOOLS = [
    {
        "type": "function",
        "function": {
            "name": "search_products",
            "description": "Mencari produk di katalog Tirenn Commerce berdasarkan makna/kebutuhan (semantic vector search), kategori, batas harga minimal/maksimal (dalam Rupiah penuh), dan status ketersediaan stok.",
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
                        "description": "ID kategori spesifik jika diketahui (1: Elektronik, 2: Fashion Pria, 3: Fashion Wanita, 4: Rumah Tangga, 5: Olahraga, 6: Kecantikan, 7: Makanan Sehat, 8: Buku/ATK)."
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
            "description": "Mengecek sisa stok dan harga aktual produk dari database secara real-time berdasarkan nama produk, kata kunci, atau SKU produk.",
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

CATEGORY_NAME_MAP = {
    1: "elektronik gadget",
    2: "fashion pakaian pria",
    3: "fashion pakaian wanita",
    4: "peralatan rumah tangga",
    5: "olahraga outdoor",
    6: "kecantikan perawatan wajah",
    7: "makanan minuman sehat",
    8: "buku alat tulis kantor"
}

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

class ChatShopperService:
    def __init__(self):
        self.ollama_url = settings.OLLAMA_BASE_URL.rstrip("/")
        self.model_name = settings.LLM_MODEL

    async def execute_tool(self, name: str, args: Dict[str, Any]) -> Dict[str, Any]:
        """Execute Python tool on behalf of the LLM"""
        logger.info(f"Executing tool: {name} with arguments: {args}")

        if name == "search_products":
            query = (args.get("query") or "").strip()
            cat_id = int(args.get("category_id") or 0)
            min_price = normalize_price(args.get("min_price"))
            max_price = normalize_price(args.get("max_price"))
            in_stock = args.get("in_stock")

            # Fallback if model sent empty query
            if not query:
                query = CATEGORY_NAME_MAP.get(cat_id, "produk pakaian wanita pria")

            # 1. Run semantic vector search
            results = vector_service.search_semantic(
                query=query,
                limit=10,
                category_id=cat_id,
                score_threshold=0.50
            )

            # 2. Filter by min_price, max_price, and in_stock
            filtered = []
            for r in results:
                if min_price is not None and r.price < min_price:
                    continue
                if max_price is not None and r.price > max_price:
                    continue
                if in_stock is True and r.stock_quantity <= 0:
                    continue
                filtered.append({
                    "id": r.id,
                    "name": r.name,
                    "sku": r.sku,
                    "price": r.price,
                    "image_url": r.image_url,
                    "stock_quantity": r.stock_quantity,
                    "in_stock": r.stock_quantity > 0,
                    "score": round(r.score, 3)
                })

            # If filtered list is empty and constraints were present, try broader search
            if not filtered and (min_price is not None or max_price is not None or in_stock is not None):
                broader_results = vector_service.search_semantic(
                    query=query,
                    limit=15,
                    category_id=cat_id,
                    score_threshold=0.35
                )
                for r in broader_results:
                    if min_price is not None and r.price < min_price:
                        continue
                    if max_price is not None and r.price > max_price:
                        continue
                    if in_stock is True and r.stock_quantity <= 0:
                        continue
                    filtered.append({
                        "id": r.id,
                        "name": r.name,
                        "sku": r.sku,
                        "price": r.price,
                        "image_url": r.image_url,
                        "stock_quantity": r.stock_quantity,
                        "in_stock": r.stock_quantity > 0,
                        "score": round(r.score, 3)
                    })

            # Return tool output for LLM (without image_url in text context so model doesn't output markdown images)
            llm_products = [
                {
                    "id": p["id"],
                    "name": p["name"],
                    "sku": p["sku"],
                    "price": p["price"],
                    "stock_quantity": p["stock_quantity"],
                    "in_stock": p["in_stock"]
                }
                for p in filtered
            ]

            return {"found_count": len(filtered), "products": llm_products, "_full_products": filtered}

        elif name == "check_product_stock":
            p_id = args.get("product_id")
            query = args.get("product_name_or_query", "")

            # If product_id is missing or looks like a hallucination (>1000), resolve via vector search
            if (not p_id or int(p_id) > 1000) and query:
                candidates = vector_service.search_semantic(query, limit=1, score_threshold=0.50)
                if candidates:
                    p_id = candidates[0].id

            # If still no ID but we have product_id, try using it
            if not p_id and not query:
                return {"error": "Mohon sebutkan nama produk atau SKU yang ingin dicek stoknya."}

            if not p_id and query:
                candidates = vector_service.search_semantic(query, limit=1, score_threshold=0.50)
                if candidates:
                    p_id = candidates[0].id

            if not p_id:
                return {"error": f"Produk '{query}' tidak ditemukan di katalog."}

            try:
                async with httpx.AsyncClient(timeout=4.0) as client:
                    resp = await client.get(f"{settings.BACKEND_API_URL}/products/{p_id}")
                    if resp.status_code == 200:
                        data = resp.json().get("data", {})
                        return {
                            "product_id": data.get("id"),
                            "name": data.get("name"),
                            "sku": data.get("sku"),
                            "stock_quantity": data.get("stock_quantity", 0),
                            "price": data.get("price"),
                            "image_url": data.get("image_url", ""),
                            "in_stock": data.get("stock_quantity", 0) > 0,
                            "badge": data.get("badge")
                        }
                    return {"error": f"Produk dengan SKU/ID {p_id} tidak ditemukan di database."}
            except Exception as e:
                return {"error": f"Gagal mengecek stok: {str(e)}"}

        elif name == "add_to_cart":
            p_id = args.get("product_id")
            query = args.get("product_name_or_query", "")
            qty = args.get("quantity", 1)

            if (not p_id or int(p_id) > 1000) and query:
                candidates = vector_service.search_semantic(query, limit=1, score_threshold=0.50)
                if candidates:
                    p_id = candidates[0].id

            if not p_id and query:
                candidates = vector_service.search_semantic(query, limit=1, score_threshold=0.50)
                if candidates:
                    p_id = candidates[0].id

            if not p_id:
                return {"error": "Produk tidak ditemukan"}

            try:
                async with httpx.AsyncClient(timeout=4.0) as client:
                    resp = await client.get(f"{settings.BACKEND_API_URL}/products/{p_id}")
                    if resp.status_code == 200:
                        data = resp.json().get("data", {})
                        return {
                            "action": "cart_added",
                            "product_id": data.get("id"),
                            "sku": data.get("sku"),
                            "name": data.get("name"),
                            "quantity": qty,
                            "price": data.get("price"),
                            "image_url": data.get("image_url"),
                            "stock_quantity": data.get("stock_quantity"),
                            "success": True
                        }
                    return {"error": "Produk tidak ditemukan"}
            except Exception as e:
                return {"error": str(e)}

        return {"error": f"Tool '{name}' tidak dikenal"}

    async def chat(self, messages: List[Dict[str, str]]) -> Dict[str, Any]:
        """Main conversational loop with Ollama & Tool Calling with Full Audit Logging"""
        start_time = time.perf_counter()
        last_user_msg = messages[-1]["content"] if messages else ""

        formatted_messages = [{"role": "system", "content": SYSTEM_PROMPT}] + messages

        executed_tools_data: List[Dict[str, Any]] = []
        suggested_products: List[Dict[str, Any]] = []
        cart_action: Optional[Dict[str, Any]] = None
        final_reply = ""

        try:
            async with httpx.AsyncClient(timeout=45.0) as client:
                # 1. Send conversation with tools to Ollama
                payload = {
                    "model": self.model_name,
                    "messages": formatted_messages,
                    "tools": SHOPPER_TOOLS,
                    "stream": False,
                    "options": {
                        "temperature": 0.3
                    }
                }

                logger.info(f"Sending chat request to Ollama: {self.ollama_url}/api/chat...")
                resp = await client.post(f"{self.ollama_url}/api/chat", json=payload)

                if resp.status_code != 200:
                    logger.warning(f"Ollama returned {resp.status_code}: {resp.text}. Falling back to search.")
                    results = vector_service.search_semantic(last_user_msg, limit=4)
                    final_reply = f"Berikut adalah rekomendasi produk terbaik untuk '{last_user_msg}':"
                    suggested_products = [
                        {"id": r.id, "name": r.name, "price": r.price, "sku": r.sku, "image_url": r.image_url} for r in results
                    ]
                else:
                    res_json = resp.json()
                    msg = res_json.get("message", {})
                    tool_calls = msg.get("tool_calls", [])

                    # 2. If model decided to call tools, execute them and send results back
                    if tool_calls:
                        formatted_messages.append(msg)

                        for tool in tool_calls:
                            fn = tool.get("function", {})
                            fn_name = fn.get("name")
                            fn_args = fn.get("arguments", {})
                            if isinstance(fn_args, str):
                                try:
                                    fn_args = json.loads(fn_args)
                                except Exception:
                                    fn_args = {}

                            tool_output = await self.execute_tool(fn_name, fn_args)
                            executed_tools_data.append({"tool": fn_name, "args": fn_args, "output": tool_output})

                            if fn_name == "search_products":
                                full_prods = tool_output.get("_full_products") or tool_output.get("products", [])
                                suggested_products.extend(full_prods)
                            elif fn_name == "check_product_stock" and "product_id" in tool_output:
                                suggested_products.append({
                                    "id": tool_output.get("product_id"),
                                    "name": tool_output.get("name"),
                                    "price": tool_output.get("price"),
                                    "sku": tool_output.get("sku"),
                                    "image_url": tool_output.get("image_url", ""),
                                })
                            elif fn_name == "add_to_cart" and tool_output.get("success"):
                                cart_action = tool_output

                            # Pass clean output without _full_products to LLM
                            clean_output = {k: v for k, v in tool_output.items() if k != "_full_products"}
                            formatted_messages.append({
                                "role": "tool",
                                "content": json.dumps(clean_output)
                            })

                        # Second turn: Ask Ollama to synthesize the final friendly response
                        second_payload = {
                            "model": self.model_name,
                            "messages": formatted_messages,
                            "stream": False,
                            "options": {
                                "temperature": 0.4
                            }
                        }
                        second_resp = await client.post(f"{self.ollama_url}/api/chat", json=second_payload)
                        if second_resp.status_code == 200:
                            final_reply = second_resp.json().get("message", {}).get("content", "")
                    else:
                        final_reply = msg.get("content", "Ada yang bisa saya bantu untuk belanja hari ini?")

        except Exception as e:
            logger.error(f"Chat error: {e}", exc_info=True)
            results = vector_service.search_semantic(last_user_msg, limit=3)
            final_reply = f"Maaf, AI Shopper sedang memproses permintaan. Berikut produk terkait '{last_user_msg}':"
            suggested_products = [
                {"id": r.id, "name": r.name, "price": r.price, "sku": r.sku, "image_url": r.image_url} for r in results
            ]

        # Calculate Latency & Write Full Structured Audit Log (Ingested by Promtail/Loki)
        elapsed_ms = (time.perf_counter() - start_time) * 1000.0
        log_entry = {
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "service": "tirenn-ai-service",
            "event": "ai_shopper_chat",
            "model": self.model_name,
            "user_prompt": last_user_msg,
            "ai_response": final_reply,
            "tool_calls_count": len(executed_tools_data),
            "tool_calls": executed_tools_data,
            "suggested_products_count": len(suggested_products),
            "latency_ms": round(elapsed_ms, 2)
        }
        logger.info(f"AI_RESPONSE_LOG: {json.dumps(log_entry, ensure_ascii=False)}")

        return {
            "reply": final_reply,
            "tool_calls": executed_tools_data,
            "suggested_products": suggested_products,
            "cart_action": cart_action
        }

shopper_service = ChatShopperService()
