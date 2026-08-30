import logging
from typing import List, Optional, Dict, Any

from app.core.config import settings
from app.domain.chat import ChatMessage, ChatShopperResult
from app.repositories.llm_repository import LLMRepository
from app.repositories.product_repository import ProductRepository
from app.repositories.analytics_repository import AnalyticsRepository
from app.usecases.knowledge_usecase import KnowledgeUseCase
from app.repositories.session_repository import SessionRepository
from app.harness.agent import AgentHarness
from app.harness.tools.admin import (
    GetExecutiveDashboardMetricsTool,
    GetRecentOrdersOverviewTool,
    GetLowStockProductsTool,
    AdjustProductStockTool,
    SearchAdminInternalSOPTool,
)

logger = logging.getLogger("ai-service.usecase.admin")

ADMIN_SYSTEM_PROMPT = """You are 'Tirenn Admin AI Copilot', an intelligent, secure, and executive operations assistant for Tirenn Commerce Merchant & Store Administration.

CORE RESPONSIBILITIES:
1. EXECUTIVE BUSINESS INTELLIGENCE:
   - Provide concise financial summaries, revenue metrics, order volumes, customer numbers, and sales trends using `get_executive_dashboard_metrics` and `get_recent_orders_overview`.
   - Format numbers clearly in Rupiah (e.g. `Rp 15.450.000`) or USD (`$1,250.00`).

2. INVENTORY & STOCK OPERATIONS (2-STEP CONFIRMATION):
   - Identify low stock products using `get_low_stock_products`.
   - Modifying inventory stock impacts real-world warehouse and database inventory. You MUST follow a strict 2-step confirmation workflow:
     * STEP 1 (Proposal & Preview): When the admin asks to change/adjust stock (e.g., "tambah stok", "kurangin stock", "set stock"), call `adjust_product_stock` with `confirmed=false`. Present the proposed details clearly: Product Name, SKU, Operation Type, Current Stock, Projected New Stock, and Audit Reason. Ask for the Admin's explicit confirmation.
     * STEP 2 (Execution): When the admin confirms or agrees to the adjustment (in any language or phrasing, e.g. "ok", "oke", "ya", "yes", "proceed", "lakukan", "setuju", "proses", "sure", etc.), YOU MUST EXECUTE the tool `adjust_product_stock` with `confirmed=true` using the exact SKU, adjustment type, quantity, and reason from the proposal.
     * If the admin cancels or disagrees (e.g. "batal", "cancel", "tidak"), acknowledge that the adjustment was cancelled without modifying any stock.
     * CRITICAL: Never claim or state that the stock has been updated without physically executing `adjust_product_stock` with `confirmed=true` and receiving the result.

3. CONFIDENTIAL WAREHOUSE & ADMIN SOP (RAG):
   - Consult internal merchant operations, warehouse picking/packing guidelines, stock audit protocols, and courier escalation rules using `search_admin_internal_sop`.
   - Quote relevant sections accurately with document titles and page numbers.

4. BILINGUAL LANGUAGE POLICY:
   - Automatically detect and match the user's language from context.
   - If the admin communicates in BAHASA INDONESIA, respond 100% in professional Bahasa Indonesia.
   - If the admin communicates in ENGLISH, respond 100% in professional English.

5. SECURITY & ROLE INTEGRITY:
   - You are exclusively accessible by authenticated store administrators.
   - Never mutate inventory without explicit admin approval.
   - Always confirm executed actions clearly with SKU, new stock quantity, and audit reason.
"""


class AdminUseCase:
    """Enterprise Admin AI Copilot UseCase with strict tool isolation and direct repository access"""

    def __init__(
        self,
        llm_repo: LLMRepository,
        product_repo: ProductRepository,
        knowledge_usecase: KnowledgeUseCase,
        analytics_repo: Optional[AnalyticsRepository] = None,
        session_repo: Optional[SessionRepository] = None
    ):
        self.llm_repo = llm_repo
        self.product_repo = product_repo
        self.analytics_repo = analytics_repo or AnalyticsRepository()
        self.knowledge_usecase = knowledge_usecase
        self.session_repo = session_repo or SessionRepository()

        # Initialize Admin-exclusive tools with direct repository injection (Zero HTTP API calls)
        self.tools = [
            GetExecutiveDashboardMetricsTool(analytics_repo=self.analytics_repo),
            GetRecentOrdersOverviewTool(analytics_repo=self.analytics_repo),
            GetLowStockProductsTool(product_repo=self.product_repo),
            AdjustProductStockTool(product_repo=self.product_repo),
            SearchAdminInternalSOPTool(knowledge_usecase=self.knowledge_usecase),
        ]

        self.agent = AgentHarness(
            llm_repo=self.llm_repo,
            tools=self.tools,
            system_prompt=ADMIN_SYSTEM_PROMPT,
            max_iterations=5
        )

    async def chat(
        self,
        messages: List[ChatMessage],
        admin_claims: Dict[str, Any],
        session_id: Optional[str] = None,
        token: Optional[str] = None
    ) -> Dict[str, Any]:
        """Execute Admin AI Copilot conversation turn with Redis sliding window history and native LLM reasoning"""
        if not messages:
            return {"reply": "Halo Admin! Ada yang bisa saya bantu terkait operasional toko, metrik omzet, stok, atau SOP gudang?", "tool_calls": []}

        latest_user_message = messages[-1]
        active_session_id = session_id or f"admin_session_default_{admin_claims.get('sub', 1)}"
        user_content = latest_user_message.content if hasattr(latest_user_message, 'content') else latest_user_message.get("content", "")

        # 1. Fetch past conversation history from Redis List
        past_history = self.session_repo.get_history(session_id=active_session_id, limit=settings.SESSION_HISTORY_LIMIT)

        # 2. Build full message sequence
        conversation_sequence: List[Dict[str, str]] = []
        for h in past_history:
            role = h.role if hasattr(h, 'role') else h.get("role", "user")
            content = h.content if hasattr(h, 'content') else h.get("content", "")
            conversation_sequence.append({"role": role, "content": content})

        # Append incoming user message if not already trailing
        user_role = latest_user_message.role if hasattr(latest_user_message, 'role') else latest_user_message.get("role", "user")
        if not conversation_sequence or conversation_sequence[-1]["content"] != user_content:
            conversation_sequence.append({"role": user_role, "content": user_content})

        # 3. Prepare execution context with Admin JWT token
        context = {
            "is_admin": True,
            "admin_email": admin_claims.get("email", "admin@gocommerce.com"),
            "admin_id": admin_claims.get("sub", 1),
            "token": token or ""
        }

        # 4. Run Admin Agent Harness (Model determines tool calls naturally from conversation history)
        agent_result = await self.agent.run(
            messages=conversation_sequence,
            context=context
        )

        reply_text = getattr(agent_result, "reply", "") or ""
        tool_calls = getattr(agent_result, "tool_calls", []) or []

        # 5. Atomically persist conversation pair to Redis
        self.session_repo.append_messages(
            session_id=active_session_id,
            messages=[
                {"role": "user", "content": user_content},
                {"role": "assistant", "content": reply_text}
            ]
        )

        return {
            "reply": reply_text,
            "tool_calls": tool_calls,
            "session_id": active_session_id,
            "admin_email": admin_claims.get("email")
        }
