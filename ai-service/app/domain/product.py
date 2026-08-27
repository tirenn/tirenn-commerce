from typing import Optional
from pydantic import BaseModel

class Product(BaseModel):
    id: int
    name: str
    sku: str
    category_id: int
    price: float
    image_url: str = ""
    stock_quantity: int = 0
    badge: str = ""
    description: str = ""
    is_active: bool = True

class ScoredProduct(BaseModel):
    id: int
    name: str
    sku: str
    category_id: int
    price: float
    image_url: str = ""
    stock_quantity: int = 0
    score: float = 0.0

class ProductIndexItem(BaseModel):
    id: int
    name: str
    category_id: int
    category_name: Optional[str] = ""
    sku: str = ""
    description: Optional[str] = ""
    price: float = 0.0
    image_url: Optional[str] = ""
    badge: Optional[str] = ""
    rating: Optional[float] = 5.0
    stock_quantity: Optional[int] = 0
