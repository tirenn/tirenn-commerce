from typing import List, Dict, Any, Optional
from pydantic import BaseModel

class ChatMessage(BaseModel):
    role: str
    content: str

class ToolCallRecord(BaseModel):
    name: str
    args: Dict[str, Any]
    output: Dict[str, Any]

class ChatShopperResult(BaseModel):
    reply: str
    tool_calls: List[Dict[str, Any]] = []
    suggested_products: List[Dict[str, Any]] = []
    cart_action: Optional[Dict[str, Any]] = None
    latency_ms: float = 0.0
