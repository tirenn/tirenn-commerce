import httpx
import logging
from typing import List, Dict, Any, Optional
from pydantic import BaseModel
from fastapi import APIRouter, HTTPException
from app.core.config import settings
from app.schemas.search import (
    IndexProductsRequest,
    IndexProductsResponse,
    SemanticSearchRequest,
    SemanticSearchResponse,
    ProductIndexItem,
)
from app.services.vector_search import vector_service
from app.services.chat_shopper import shopper_service

logger = logging.getLogger("ai-service.api")

router = APIRouter()

class ChatMessage(BaseModel):
    role: str
    content: str

class ChatShopperRequest(BaseModel):
    messages: List[ChatMessage]

class ChatShopperResponse(BaseModel):
    success: bool
    reply: str
    tool_calls: List[Dict[str, Any]] = []
    suggested_products: List[Dict[str, Any]] = []
    cart_action: Optional[Dict[str, Any]] = None

@router.post("/chat/shopper", response_model=ChatShopperResponse)
async def chat_shopper(req: ChatShopperRequest):
    """Conversational AI Shopper with Ollama Qwen 2.5 and Agentic Tool Calling"""
    try:
        messages_dict = [{"role": m.role, "content": m.content} for m in req.messages]
        result = await shopper_service.chat(messages_dict)
        return ChatShopperResponse(
            success=True,
            reply=result.get("reply", ""),
            tool_calls=result.get("tool_calls", []),
            suggested_products=result.get("suggested_products", []),
            cart_action=result.get("cart_action"),
        )
    except Exception as e:
        logger.error(f"Chat shopper error: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/index-products", response_model=IndexProductsResponse)
async def index_products(req: IndexProductsRequest):
    """Index or update a batch of products into the vector store"""
    try:
        count = vector_service.index_products(req.products)
        return IndexProductsResponse(
            success=True,
            message=f"Indexed {count} products successfully",
            indexed_count=count
        )
    except Exception as e:
        logger.error(f"Failed to index products: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/search/semantic", response_model=SemanticSearchResponse)
async def semantic_search(req: SemanticSearchRequest):
    """Perform high-precision AI semantic search across indexed catalog"""
    try:
        results = vector_service.search_semantic(
            query=req.query,
            limit=req.limit,
            category_id=req.category_id or 0
        )
        return SemanticSearchResponse(
            success=True,
            query=req.query,
            total_results=len(results),
            data=results
        )
    except Exception as e:
        logger.error(f"Semantic search failed: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/sync-from-backend")
async def sync_from_backend():
    """Fetch all products from Go backend and index them in vector store"""
    try:
        all_items: list[ProductIndexItem] = []
        page = 1
        limit = 50

        async with httpx.AsyncClient(timeout=10.0) as client:
            while True:
                resp = await client.get(f"{settings.BACKEND_API_URL}/products?page={page}&limit={limit}")
                if resp.status_code != 200:
                    break
                
                data = resp.json()
                products_raw = data.get("data", [])
                if not products_raw or not isinstance(products_raw, list):
                    break

                for p in products_raw:
                    cat = p.get("category") or {}
                    all_items.append(
                        ProductIndexItem(
                            id=p.get("id"),
                            name=p.get("name", ""),
                            category_id=p.get("category_id", 0),
                            category_name=cat.get("name", ""),
                            sku=p.get("sku", ""),
                            description=p.get("description", ""),
                            price=float(p.get("price", 0.0)),
                            image_url=p.get("image_url", ""),
                            badge=p.get("badge", ""),
                            rating=float(p.get("rating", 5.0)),
                            stock_quantity=int(p.get("stock_quantity", 0)),
                        )
                    )

                meta = data.get("meta") or {}
                total_pages = meta.get("total_pages", 1)
                if page >= total_pages:
                    break
                page += 1
            
            count = vector_service.index_products(all_items)
            return {"success": True, "synced_products": count}
    except Exception as e:
        logger.error(f"Error syncing from backend: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))
