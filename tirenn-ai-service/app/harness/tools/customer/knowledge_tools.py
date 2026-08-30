import logging
from typing import Dict, Any, Optional
from app.harness.tools.base import BaseTool

logger = logging.getLogger("ai-service.harness.tools.customer.knowledge")


class SearchStorePoliciesAndSOPTool(BaseTool):
    """Tool for querying customer-facing shopping SOP, warranty, returns, and delivery SLA documents"""

    name = "search_store_policies_and_sop"
    description = "Search customer-facing store policies, customer buying guide SOP, return/warranty terms, and shipping SLA from official customer knowledge documents."
    parameters_schema = {
        "type": "object",
        "properties": {
            "query": {
                "type": "string",
                "description": "Question or topic regarding customer shopping SOP, buying instructions, warranty, refund/returns, or shipping SLA (e.g. 'cara retur barang cacat', 'kebijakan garansi', 'SLA waktu pengiriman'). Required."
            }
        },
        "required": ["query"]
    }

    def __init__(self, knowledge_usecase):
        self.knowledge_usecase = knowledge_usecase

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        query = (args.get("query") or "").strip()
        logger.info(f"📚 [CUSTOMER TOOL: search_store_policies_and_sop] query='{query}' (doc_type='SOP_CUSTOMER')")

        if not self.knowledge_usecase:
            return {"status": "error", "message": "Knowledge Base is not initialized."}

        results = self.knowledge_usecase.query_knowledge(
            query=query,
            limit=3,
            score_threshold=0.15,
            doc_type="SOP_CUSTOMER"
        )
        if not results:
            return {"status": "not_found", "message": f"No customer store policy or SOP found for '{query}'."}

        formatted = [
            {
                "document": r.get("document_title"),
                "page": r.get("page_number"),
                "content": f"<untrusted_document_content>\n{r.get('content', '').strip()}\n</untrusted_document_content>",
                "relevance_score": round(r.get("score", 0.0), 3)
            }
            for r in results
        ]

        return {
            "status": "found",
            "found_count": len(formatted),
            "policy_excerpts": formatted
        }
