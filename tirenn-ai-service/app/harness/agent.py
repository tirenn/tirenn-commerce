import time
import json
import logging
from typing import List, Dict, Any, Optional

from app.domain.chat import ChatMessage, ChatShopperResult
from app.repositories.llm_repository import LLMRepository
from app.harness.tools.base import BaseTool
from app.harness.guardrails.relevance import RelevanceGuardrail
from app.harness.guardrails.safety import SafetyGuardrail

logger = logging.getLogger("ai-service.harness.agent")

class AgentHarness:
    """Enterprise ReAct Agent Harness for Tirenn Commerce AI Services"""

    def __init__(
        self,
        llm_repo: LLMRepository,
        tools: List[BaseTool],
        system_prompt: str,
        max_iterations: int = 5
    ):
        self.llm_repo = llm_repo
        self.tools_map: Dict[str, BaseTool] = {t.name: t for t in tools}
        self.system_prompt = system_prompt
        self.relevance_guardrail = RelevanceGuardrail()
        self.safety_guardrail = SafetyGuardrail(max_iterations=max_iterations)

    def get_openai_tools_schema(self) -> List[Dict[str, Any]]:
        """Extract OpenAPI/Ollama tool schemas from registered tools"""
        return [tool.to_openai_tool_schema() for tool in self.tools_map.values()]

    async def run(
        self,
        messages: List[ChatMessage],
        context: Optional[Dict[str, Any]] = None
    ) -> ChatShopperResult:
        """Run full autonomous execution turn with tool dispatch, observation guardrails, and relevance pruning"""
        start_time = time.perf_counter()
        context = context or {}
        user_name = context.get("user_name")
        is_authenticated = context.get("is_authenticated", False)
        last_user_msg = messages[-1].content if messages else ""

        # 1. Perception & System Prompt Construction
        system_text = self.system_prompt
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
            tools_schema = self.get_openai_tools_schema()

            # 2. Turn 1: Reason & Select Actions
            assistant_msg = await self.llm_repo.chat(
                messages=formatted_messages,
                tools=tools_schema,
                temperature=0.0
            )

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

                    tool_instance = self.tools_map.get(tool_name)
                    tool_start = time.perf_counter()

                    if tool_instance:
                        tool_result = await tool_instance.execute(tool_args, context=context)

                        # --- OBSERVATION & RELEVANCE GUARDRAIL STEP ---
                        if tool_name == "search_products":
                            full_prods = tool_result.pop("_full_products", [])
                            raw_query = tool_result.pop("_raw_query", last_user_msg)

                            # Apply Intelligent Relevance Filter
                            verified_full_prods = self.relevance_guardrail.filter_products(
                                query=raw_query or last_user_msg,
                                products=full_prods
                            )

                            # Sync filtered products back into tool result for LLM context
                            pruned_llm_products = [
                                {
                                    "id": p["id"],
                                    "name": p["name"],
                                    "sku": p["sku"],
                                    "price": p["price"],
                                    "currency": p.get("currency", "IDR"),
                                    "stock_quantity": p["stock_quantity"],
                                    "in_stock": p["in_stock"]
                                }
                                for p in verified_full_prods
                            ]
                            tool_result["products"] = pruned_llm_products
                            tool_result["found_count"] = len(pruned_llm_products)

                            suggested_products.extend(verified_full_prods)

                        elif tool_name == "check_product_stock":
                            full_prod = tool_result.pop("_full_product", None)
                            if full_prod:
                                suggested_products.append(full_prod)

                        elif tool_name == "add_to_cart":
                            full_prod = tool_result.pop("_full_product", None)
                            if full_prod:
                                suggested_products.append(full_prod)
                            cart_action = tool_result
                    else:
                        tool_result = {"error": f"Tool '{tool_name}' not found in registry."}

                    tool_dur_ms = (time.perf_counter() - tool_start) * 1000.0
                    logger.info(
                        f"⚡ [HARNESS_TOOL_EXECUTED] name='{tool_name}' | "
                        f"latency={tool_dur_ms:.1f}ms | status='success'"
                    )

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

                # 3. Turn 2: Synthesize Grounded Natural Language Response
                followup_msg = await self.llm_repo.chat(
                    messages=formatted_messages,
                    temperature=0.3
                )
                final_reply = followup_msg.get("content", "")

            else:
                # Direct response without tool call
                final_reply = assistant_msg.get("content", "")

                # Intent fallback: If user query has product search intent, invoke search tool through harness
                search_triggers = [
                    "cari", "rekomendasi", "produk", "barang", "harga", "stok", "tas", "baju", "sepatu",
                    "celana", "kopi", "kaos", "hoodie", "dompet", "jaket", "sandal", "pakaian",
                    "elektronik", "headphone", "jam", "dress", "rok", "blouse", "beli", "ada", "mau", "diskon",
                    "find", "search", "recommend", "show", "product", "shoes", "shirt", "pants", "bag",
                    "coffee", "tea", "skincare", "perfume", "price", "stock", "cheap", "best", "buy"
                ]
                if any(w in last_user_msg.lower() for w in search_triggers):
                    search_tool = self.tools_map.get("search_products")
                    if search_tool:
                        fallback_res = await search_tool.execute(args={"query": last_user_msg}, context=context)
                        full_prods = fallback_res.pop("_full_products", [])
                        raw_q = fallback_res.pop("_raw_query", last_user_msg)

                        verified_prods = self.relevance_guardrail.filter_products(
                            query=raw_q or last_user_msg,
                            products=full_prods
                        )

                        if verified_prods:
                            suggested_products = verified_prods
                            pruned_llm_products = [
                                {
                                    "id": p["id"],
                                    "name": p["name"],
                                    "sku": p["sku"],
                                    "price": p["price"],
                                    "currency": p.get("currency", "IDR"),
                                    "stock_quantity": p["stock_quantity"],
                                    "in_stock": p["in_stock"]
                                }
                                for p in verified_prods
                            ]
                            fallback_res["products"] = pruned_llm_products
                            fallback_res["found_count"] = len(pruned_llm_products)

                            executed_tools_data.append({
                                "name": "search_products",
                                "params": {"query": last_user_msg},
                                "status": "success",
                                "result": fallback_res
                            })

                            if not final_reply or len(final_reply.strip()) < 10:
                                is_en = any(w in last_user_msg.lower() for w in ["find", "search", "recommend", "show", "product", "shoes", "shirt", "bag"])
                                if is_en:
                                    final_reply = f"Here are the best verified matching products for '{last_user_msg}' available at Tirenn Commerce:"
                                else:
                                    final_reply = f"Berikut adalah rekomendasi produk yang terverifikasi sesuai untuk '{last_user_msg}' di Tirenn Commerce:"

                # Universal Fallback: If no products retrieved yet and query has substance, execute search tool
                if not suggested_products and len(last_user_msg.strip()) >= 2:
                    search_tool = self.tools_map.get("search_products")
                    if search_tool:
                        fallback_res = await search_tool.execute(args={"query": last_user_msg}, context=context)
                        full_prods = fallback_res.pop("_full_products", [])
                        raw_q = fallback_res.pop("_raw_query", last_user_msg)

                        verified_prods = self.relevance_guardrail.filter_products(
                            query=raw_q or last_user_msg,
                            products=full_prods
                        )

                        if verified_prods:
                            suggested_products = verified_prods
                            pruned_llm_products = [
                                {
                                    "id": p["id"],
                                    "name": p["name"],
                                    "sku": p["sku"],
                                    "price": p["price"],
                                    "currency": p.get("currency", "IDR"),
                                    "stock_quantity": p["stock_quantity"],
                                    "in_stock": p["in_stock"]
                                }
                                for p in verified_prods
                            ]
                            fallback_res["products"] = pruned_llm_products
                            fallback_res["found_count"] = len(pruned_llm_products)

                            executed_tools_data.append({
                                "name": "search_products",
                                "params": {"query": last_user_msg},
                                "status": "success",
                                "result": fallback_res
                            })

                            if not final_reply or len(final_reply.strip()) < 10:
                                is_en = any(w in last_user_msg.lower() for w in ["find", "search", "recommend", "show", "product", "shoes", "shirt", "bag"])
                                if is_en:
                                    final_reply = f"Here are the products found in our catalog for '{last_user_msg}':"
                                else:
                                    final_reply = f"Berikut produk yang ditemukan di katalog untuk '{last_user_msg}':"

        except Exception as e:
            logger.error(f"AgentHarness execution error: {e}", exc_info=True)
            final_reply = "Maaf, terjadi kendala saat memproses permintaan belanja Anda. Silakan coba kembali."

        # Sanitize final output
        final_reply = self.safety_guardrail.sanitize_output_text(final_reply)

        # Deduplicate visual product cards by ID
        unique_prods: List[Dict[str, Any]] = []
        seen_ids = set()
        for p in suggested_products:
            if p.get("id") and p["id"] not in seen_ids:
                seen_ids.add(p["id"])
                unique_prods.append(p)

        elapsed_ms = (time.perf_counter() - start_time) * 1000.0

        return ChatShopperResult(
            reply=final_reply,
            tool_calls=executed_tools_data,
            suggested_products=unique_prods,
            cart_action=cart_action,
            latency_ms=elapsed_ms
        )
