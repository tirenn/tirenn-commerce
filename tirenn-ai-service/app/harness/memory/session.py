import time
from typing import List, Dict, Any, Optional

class SessionMemory:
    """Manages short-term conversation state, execution history, and context compaction"""

    def __init__(self, max_history_turns: int = 10):
        self.max_history_turns = max_history_turns
        self.turns: List[Dict[str, Any]] = []

    def add_turn(self, role: str, content: str, tool_calls: Optional[List[Dict[str, Any]]] = None):
        turn_entry = {
            "role": role,
            "content": content,
            "timestamp": time.time()
        }
        if tool_calls:
            turn_entry["tool_calls"] = tool_calls
        self.turns.append(turn_entry)
        if len(self.turns) > self.max_history_turns * 2:
            self.turns = self.turns[-(self.max_history_turns * 2):]

    def get_messages_for_llm(self, system_prompt: str) -> List[Dict[str, Any]]:
        messages = [{"role": "system", "content": system_prompt}]
        for t in self.turns:
            msg = {"role": t["role"], "content": t["content"]}
            if "tool_calls" in t:
                msg["tool_calls"] = t["tool_calls"]
            messages.append(msg)
        return messages
