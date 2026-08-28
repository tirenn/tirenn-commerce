import re
import logging
from typing import List, Dict, Any, Set, Tuple

logger = logging.getLogger("ai-service.harness.guardrails.relevance")

# Mutually exclusive attribute pairs (Concept A vs Concept B)
# If user query matches Concept A, products matching Concept B must be rejected, and vice versa.
CONTRADICTORY_PAIRS: List[Tuple[Set[str], Set[str], str]] = [
    # 1. Long vs Short (e.g. Celana Panjang vs Celana Pendek / Shorts)
    (
        {"panjang", "long", "trousers"},
        {"pendek", "short", "shorts", "bermuda", "boxer"},
        "Length attribute contradiction (Long vs Short)"
    ),
    # 2. Gender: Men vs Women
    (
        {"pria", "men", "man", "cowok", "laki", "gentleman"},
        {"wanita", "women", "woman", "cewek", "perempuan", "ladies", "dress", "blouse", "rok", "heels"},
        "Gender attribute contradiction (Men vs Women)"
    ),
    # 3. Product Types: Top/Shirt vs Pants/Bottoms
    (
        {"kaos", "t-shirt", "shirt", "kemeja", "hoodie", "jaket", "jacket", "blouse", "sweater"},
        {"celana", "pants", "jeans", "chino", "trousers", "shorts", "rok", "skirt"},
        "Product type contradiction (Tops vs Bottoms)"
    ),
    # 4. Product Types: Shoes/Footwear vs Clothing
    (
        {"sepatu", "shoes", "sneakers", "boots", "heels", "sandal", "sandals", "slippers"},
        {"kaos", "kemeja", "celana", "pants", "baju", "dress", "jaket"},
        "Product type contradiction (Footwear vs Clothing)"
    ),
    # 5. Food & Drink: Coffee vs Tea/Snacks
    (
        {"kopi", "coffee", "arabika", "robusta", "espresso", "cold brew"},
        {"teh", "tea", "matcha", "snack", "camilan", "keripik", "almond"},
        "Product category contradiction (Coffee vs Tea/Snack)"
    ),
    # 6. Electronics vs Clothing
    (
        {"headphone", "earphone", "earbuds", "tws", "smartwatch", "speaker", "keyboard", "mouse", "audio"},
        {"baju", "celana", "sepatu", "tas", "dompet", "dress", "kemeja"},
        "Major category contradiction (Electronics vs Apparel)"
    ),
]

class RelevanceGuardrail:
    """Intelligent Relevance & Semantic Contradiction Guardrail
    
    Inspects search returns from tool execution and removes candidate products that
    contradict the user's explicit query constraints (e.g. filtering out 'celana pendek'
    when user queries 'celana panjang').
    """

    def __init__(self):
        pass

    def filter_products(
        self,
        query: str,
        products: List[Dict[str, Any]],
        strictness: float = 0.8
    ) -> List[Dict[str, Any]]:
        """Filter out irrelevant or contradictory products from raw tool search output"""
        if not products or not query:
            return products

        clean_query = query.lower().strip()
        query_words = set(re.findall(r'\b\w+\b', clean_query))

        # Detect active constraints in user query
        excluded_concepts: List[Tuple[Set[str], str]] = []

        for group_a, group_b, reason in CONTRADICTORY_PAIRS:
            # Check if query matches group A
            if any(term in clean_query or term in query_words for term in group_a):
                excluded_concepts.append((group_b, f"{reason} -> User asked for {group_a}, rejected {group_b}"))
            # Check if query matches group B
            elif any(term in clean_query or term in query_words for term in group_b):
                excluded_concepts.append((group_a, f"{reason} -> User asked for {group_b}, rejected {group_a}"))

        filtered: List[Dict[str, Any]] = []
        removed_count = 0

        for p in products:
            p_name = (p.get("name") or "").lower()
            p_sku = (p.get("sku") or "").lower()
            p_subcat = (p.get("sub_category_name") or "").lower()
            p_text = f"{p_name} {p_sku} {p_subcat}"
            p_words = set(re.findall(r'\b\w+\b', p_text))

            is_contradictory = False
            reject_reason = ""

            for excluded_terms, reason in excluded_concepts:
                # If product contains any excluded term that wasn't in the user's original query
                matching_exclusions = [term for term in excluded_terms if term in p_text or term in p_words]
                if matching_exclusions:
                    # Make sure the user didn't explicitly ask for this term
                    if not any(m in clean_query for m in matching_exclusions):
                        is_contradictory = True
                        reject_reason = f"{reason} (Matched: {matching_exclusions})"
                        break

            if is_contradictory:
                logger.info(
                    f"🛡️ [RELEVANCE_GUARDRAIL_PRUNED] "
                    f"Product '{p.get('name')}' (ID: {p.get('id')}) was removed from search results. "
                    f"Reason: {reject_reason}"
                )
                removed_count += 1
            else:
                filtered.append(p)

        if removed_count > 0:
            logger.info(
                f"🛡️ [RELEVANCE_GUARDRAIL_SUMMARY] Query='{query}' | "
                f"Original candidates={len(products)} | "
                f"Pruned irrelevant={removed_count} | "
                f"Retained verified={len(filtered)}"
            )

        # Fallback safety: If guardrail was overly strict and removed ALL products,
        # but original products had strong scores, return top candidate to avoid empty response
        if not filtered and products:
            logger.warning(f"🛡️ Guardrail pruned all products for '{query}'. Fallback to top scored candidate.")
            filtered = [products[0]]

        return filtered
