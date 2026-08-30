import json
import time
import logging
from typing import List, Dict, Any, Optional
from app.core.config import settings
from app.repositories.llm_repository import LLMRepository
from app.domain.chat import ChatShopperResult, ChatMessage
from app.harness.tools.base import BaseTool

logger = logging.getLogger("ai-service.harness.agent")

class AgentHarness:
    """Domain-Agnostic ReAct Agent Harness with Native LLM Tool Calling and Multilingual Reasoning"""

    def __init__(
        self,
        llm_repo: LLMRepository,
        tools: List[BaseTool],
        system_prompt: str,
        max_iterations: int = 5,
    ):
        self.llm_repo = llm_repo
        self.tools = tools
        self.tools_map = {tool.name: tool for tool in tools}
        self.system_prompt = system_prompt
        self.max_iterations = max_iterations

    def get_openai_tools_schema(self) -> List[Dict[str, Any]]:
        """Export all registered tools in standard OpenAI / Ollama function calling schema"""
        return [tool.to_openai_schema() for tool in self.tools]

    async def run(
        self,
        messages: List[ChatMessage],
        context: Optional[Dict[str, Any]] = None
    ) -> ChatShopperResult:
        """Execute Canonical ReAct Loop (Reason -> Action -> Observation -> Synthesis)"""
        start_time = time.perf_counter()
        context = context or {}
        context["messages"] = messages

        # Construct system prompt and full conversational history
        system_text = self.system_prompt
        user_name = context.get("user_name")
        if user_name:
            system_text += f"\nCustomer Context: Customer name is '{user_name}' (authenticated user)."

        formatted_messages = [{"role": "system", "content": system_text}] + [
            {
                "role": m.role if hasattr(m, 'role') else m.get('role', 'user'),
                "content": m.content if hasattr(m, 'content') else m.get('content', '')
            }
            for m in messages
        ]

        executed_tools_data: List[Dict[str, Any]] = []
        suggested_products: List[Dict[str, Any]] = []
        cart_action: Optional[Dict[str, Any]] = None
        final_reply = ""

        try:
            tools_schema = self.get_openai_tools_schema()

            for iteration in range(self.max_iterations):
                # 1. LLM Reasoning & Action Selection
                assistant_msg = await self.llm_repo.chat(
                    messages=formatted_messages,
                    tools=tools_schema,
                    temperature=settings.LLM_TOOL_TEMPERATURE
                )

                tool_calls = assistant_msg.get("tool_calls", [])

                if not tool_calls:
                    # Model produced a natural language answer -> Turn complete
                    final_reply = assistant_msg.get("content", "")
                    break

                # 2. Append model's tool_calls turn to history
                formatted_messages.append(assistant_msg)

                # 3. Tool Execution & Observation
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
                        tool_args = raw_args or {}

                    tool_instance = self.tools_map.get(tool_name)
                    tool_start = time.perf_counter()

                    if tool_instance:
                        tool_result = await tool_instance.execute(tool_args, context=context)

                        if tool_name == "search_products":
                            full_prods = tool_result.pop("_full_products", [])
                            tool_result.pop("_raw_query", None)
                            
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
                                for p in full_prods[:settings.CHAT_SEARCH_LIMIT]
                            ]
                            tool_result["products"] = pruned_llm_products
                            tool_result["found_count"] = len(pruned_llm_products)
                            suggested_products.extend(full_prods[:settings.CHAT_SEARCH_LIMIT])

                        elif tool_name in ["get_product_detail", "get_product_stock"]:
                            full_prod = tool_result.pop("_full_product", None)
                            if full_prod:
                                suggested_products.append(full_prod)

                        elif tool_name in ["add_to_cart", "view_cart"]:
                            full_prod = tool_result.pop("_full_product", None)
                            if full_prod:
                                suggested_products.append(full_prod)
                            cart_action = tool_result
                    else:
                        tool_result = {"error": f"Tool '{tool_name}' not found in registry."}

                    tool_dur_ms = (time.perf_counter() - tool_start) * 1000.0
                    logger.info(
                        f"⚡ [HARNESS_TOOL] iteration={iteration+1} | name='{tool_name}' | "
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
                        "content": json.dumps(tool_result, ensure_ascii=False, default=str)
                    })

            # If max iterations reached without text reply, provide clean fallback
            if not final_reply:
                final_reply = "Silakan beri tahu saya jika ada informasi lain yang Anda butuhkan."

        except Exception as e:
            logger.error(f"AgentHarness execution error: {e}", exc_info=True)
            final_reply = "Maaf, terjadi kendala saat memproses permintaan Anda. Silakan coba kembali."

        # Deduplicate product cards by ID
        unique_prods: List[Dict[str, Any]] = []
        seen_ids = set()
        for p in suggested_products:
            if p.get("id") and p["id"] not in seen_ids:
                seen_ids.add(p["id"])
                unique_prods.append(p)

        # In-Context Sync: If LLM curated products in its final reply, keep only the products LLM referenced
        if final_reply and unique_prods:
            reply_lower = final_reply.lower()
            curated_prods = [
                p for p in unique_prods
                if (p.get("sku") and p["sku"].lower() in reply_lower)
                or (p.get("name") and len(p["name"]) > 5 and p["name"].lower()[:20] in reply_lower)
            ]
            if curated_prods:
                unique_prods = curated_prods

        unique_prods = unique_prods[:settings.CHAT_SEARCH_LIMIT]
        elapsed_ms = (time.perf_counter() - start_time) * 1000.0

        return ChatShopperResult(
            reply=final_reply,
            tool_calls=executed_tools_data,
            suggested_products=unique_prods,
            cart_action=cart_action,
            latency_ms=elapsed_ms
        )
