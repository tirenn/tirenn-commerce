"""
Unit tests for Prompt-as-Code Loader (ai/app/core/prompt_loader.py)
Validates loading, caching, and error handling of Markdown system prompts.
"""

import pytest
from app.core.prompt_loader import load_prompt, clear_prompt_cache, _PROMPT_CACHE


def test_load_shopper_prompt():
    """Verify shopper_agent.md prompt loads with expected persona and instructions"""
    clear_prompt_cache()
    prompt = load_prompt("shopper_agent.md")

    assert len(prompt) > 200
    assert "Tirenn AI Shopper" in prompt
    assert "BILINGUAL LANGUAGE POLICY" in prompt
    assert "GROUNDING & IN-CONTEXT CURATION" in prompt
    assert "shopper_agent.md" in _PROMPT_CACHE


def test_load_admin_prompt():
    """Verify admin_agent.md prompt loads with 2-step confirmation workflow"""
    clear_prompt_cache()
    prompt = load_prompt("admin_agent.md")

    assert len(prompt) > 200
    assert "Tirenn Admin AI Copilot" in prompt
    assert "INVENTORY & STOCK OPERATIONS" in prompt
    assert "2-step confirmation workflow" in prompt
    assert "admin_agent.md" in _PROMPT_CACHE


def test_prompt_loader_caching():
    """Verify subsequent calls return cached in-memory string without file I/O"""
    clear_prompt_cache()
    p1 = load_prompt("shopper_agent")
    p2 = load_prompt("shopper_agent")

    assert p1 is p2, "Subsequent calls should return exact cached string reference"


def test_load_nonexistent_prompt_raises():
    """Verify loader raises FileNotFoundError when prompt file does not exist"""
    with pytest.raises(FileNotFoundError):
        load_prompt("nonexistent_agent_9999.md")
