from abc import ABC, abstractmethod
from typing import Dict, Any, Optional

class BaseTool(ABC):
    """Abstract Base Class for Agent Harness Tools"""

    name: str = ""
    description: str = ""
    parameters_schema: Dict[str, Any] = {}

    @abstractmethod
    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Execute the tool with given arguments and context"""
        pass

    def to_openai_tool_schema(self) -> Dict[str, Any]:
        """Convert tool definition to standard OpenAI/Ollama function calling schema"""
        return {
            "type": "function",
            "function": {
                "name": self.name,
                "description": self.description,
                "parameters": self.parameters_schema
            }
        }

    to_openai_schema = to_openai_tool_schema
