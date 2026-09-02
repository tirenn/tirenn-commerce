import logging
from pathlib import Path
from typing import Dict

logger = logging.getLogger("ai-service.core.prompt_loader")

_PROMPT_CACHE: Dict[str, str] = {}
_PROMPTS_DIR = Path(__file__).resolve().parent.parent / "prompts"


def load_prompt(prompt_name: str, reload: bool = False) -> str:
    """
    Loads and caches a system prompt Markdown file from ai/app/prompts/.
    Supports hot-reloading if reload=True.

    Example:
        system_prompt = load_prompt("shopper_agent.md")
    """
    filename = prompt_name if prompt_name.endswith(".md") else f"{prompt_name}.md"

    if not reload and filename in _PROMPT_CACHE:
        return _PROMPT_CACHE[filename]

    file_path = _PROMPTS_DIR / filename

    if not file_path.exists():
        logger.error(f"Prompt file not found at: {file_path}")
        raise FileNotFoundError(f"System prompt file '{filename}' does not exist in {_PROMPTS_DIR}")

    try:
        content = file_path.read_text(encoding="utf-8").strip()
        _PROMPT_CACHE[filename] = content
        logger.info(f"📄 [PROMPT_LOADED] Loaded prompt '{filename}' ({len(content)} chars)")
        return content
    except Exception as e:
        logger.error(f"Failed to read prompt file {file_path}: {e}")
        raise e


def clear_prompt_cache():
    """Clear in-memory prompt cache to force file reload"""
    _PROMPT_CACHE.clear()
    logger.info("🧹 [PROMPT_CACHE_CLEARED] In-memory prompt cache cleared.")
