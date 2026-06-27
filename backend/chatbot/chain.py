"""
chatbot/chain.py — LangChain + Ollama Chain Factory
"""

from __future__ import annotations

import logging

from langchain_ollama import ChatOllama
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
from langchain_core.messages import HumanMessage, AIMessage
from langchain_core.output_parsers import StrOutputParser

from .config import (
    OLLAMA_BASE_URL,
    OLLAMA_MODEL,
    OLLAMA_TEMPERATURE,
    OLLAMA_MAX_TOKENS,
)

logger = logging.getLogger("chatbot.chain")

_llm: ChatOllama | None = None


def _get_llm() -> ChatOllama:
    global _llm
    if _llm is None:
        logger.info("Initialising ChatOllama — model=%s base_url=%s", OLLAMA_MODEL, OLLAMA_BASE_URL)
        _llm = ChatOllama(
            model=OLLAMA_MODEL,
            base_url=OLLAMA_BASE_URL,
            temperature=OLLAMA_TEMPERATURE,
            num_predict=OLLAMA_MAX_TOKENS,
        )
    return _llm


def history_to_messages(history: list[dict[str, str]]) -> list:
    messages = []
    for msg in history:
        role    = msg.get("role", "user")
        content = msg.get("content", "")
        if role == "user":
            messages.append(HumanMessage(content=content))
        elif role == "assistant":
            messages.append(AIMessage(content=content))
    return messages


def invoke_chain(
    system_prompt: str,
    history: list[dict[str, str]],
    user_input: str,
) -> str:
    prompt = ChatPromptTemplate.from_messages([
        ("system", system_prompt),
        MessagesPlaceholder(variable_name="history"),
        ("human", "{input}"),
    ])

    chain = prompt | _get_llm() | StrOutputParser()
    lc_history = history_to_messages(history)

    try:
        reply: str = chain.invoke({"history": lc_history, "input": user_input})
    except Exception as exc:
        logger.error("Ollama invocation failed: %s", exc, exc_info=True)
        raise RuntimeError(
            "Could not reach Ollama. Run `ollama serve` and ensure the model is pulled."
        ) from exc

    if not reply or not reply.strip():
        raise RuntimeError("I could not process that. Please try asking again.")

    return reply.strip()
