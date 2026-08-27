from typing import List, Optional
from pydantic import BaseModel, Field

class ProductIndexItem(BaseModel):
    id: int
    name: str
    category_id: int
    category_name: Optional[str] = ""
    sku: str
    description: Optional[str] = ""
    price: float
    image_url: Optional[str] = ""
    badge: Optional[str] = ""
    rating: Optional[float] = 5.0
    stock_quantity: Optional[int] = 0

class IndexProductsRequest(BaseModel):
    products: List[ProductIndexItem]

class IndexProductsResponse(BaseModel):
    success: bool
    message: str
    indexed_count: int

class SemanticSearchRequest(BaseModel):
    query: str = Field(..., description="User search text or natural language prompt", min_length=1)
    limit: int = Field(default=12, ge=1, le=50)
    category_id: Optional[int] = Field(default=0, ge=0)

class ScoredProductResult(BaseModel):
    id: int
    score: float
    name: str
    category_id: int
    sku: str
    price: float
    image_url: Optional[str] = ""
    stock_quantity: Optional[int] = 0

class SemanticSearchResponse(BaseModel):
    success: bool
    query: str
    total_results: int
    data: List[ScoredProductResult]
