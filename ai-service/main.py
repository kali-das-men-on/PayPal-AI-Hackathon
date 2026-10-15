"""DisputeDesk AI service: FastAPI app served by uvicorn.

Two engines behind one endpoint:
  * LangChain + a chat model, if DD_MODEL is set (e.g. "anthropic:claude-sonnet-5-5").
  * A transparent rule-based "heuristic" engine otherwise, so the demo runs with no API key.
The response always says which engine produced it.
"""
import json
import os
from typing import Literal, Optional

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

app = FastAPI(title="DisputeDesk AI")


class EvidenceIn(BaseModel):
    carrier: Optional[str] = None
    tracking_number: Optional[str] = None
    delivery_status: str = "none"
    delivered_at: Optional[str] = None
    buyer_message: Optional[str] = None


class DisputeIn(BaseModel):
    id: str
    reason: str
    amount: float
    currency: str = "USD"
    item: str = ""
    evidence: EvidenceIn


class Analysis(BaseModel):
    category: str = Field(description="Short dispute category, e.g. item_not_received")
    recommendation: Literal["contest", "settle"]
    evidence_strength: Literal["strong", "weak", "none"]
    confidence: float = Field(ge=0, le=1)
    reasoning: str = Field(description="2-3 plain sentences a merchant can read")
    draft_response: str = Field(description="Polite message to PayPal/the buyer using ONLY the provided facts")
    engine: str = "unset"


SYSTEM = """You help a small online seller respond to PayPal disputes.
Rules: use ONLY the facts provided; never invent tracking numbers, dates or policies.
Be fair: if the evidence does not support the seller, recommend 'settle' (refund).
Recommend 'contest' only when the evidence clearly shows the order was fulfilled.
Judge evidence_strength honestly: strong = delivered with tracking; weak = partial; none = nothing."""


def heuristic(d: DisputeIn) -> Analysis:
    ev = d.evidence
    cat = "item_not_received" if "NOT_RECEIVED" in d.reason else "not_as_described"
    if ev.delivery_status == "delivered" and ev.tracking_number:
        return Analysis(
            category=cat, recommendation="contest", evidence_strength="strong", confidence=0.85,
            reasoning=f"Carrier tracking shows delivery ({ev.carrier} {ev.tracking_number}). "
                      "Proof of delivery is the key evidence PayPal looks for in this dispute type.",
            draft_response=f"Hello, order {d.id} for '{d.item}' was shipped via {ev.carrier} with tracking "
                           f"{ev.tracking_number} and marked delivered on {ev.delivered_at}. "
                           "Please review the attached tracking details. We are happy to help further.",
            engine="heuristic")
    if ev.delivery_status == "in_transit":
        return Analysis(category=cat, recommendation="contest", evidence_strength="weak", confidence=0.5,
                        reasoning="Shipment is still in transit; evidence is partial.",
                        draft_response="Your order is in transit; tracking is attached.", engine="heuristic")
    return Analysis(
        category=cat, recommendation="settle", evidence_strength="none", confidence=0.8,
        reasoning="No shipment or delivery evidence is on file, so a defense is unlikely to hold. "
                  "A prompt refund is the fair outcome.",
        draft_response=f"Hello, we're sorry about order {d.id}. We're issuing a full refund of "
                       f"{d.amount:.2f} {d.currency} right away.",
        engine="heuristic")


def langchain_analyze(d: DisputeIn, model_name: str) -> Analysis:
    # Imported lazily so the heuristic mode needs no LangChain install.
    from langchain.chat_models import init_chat_model
    from langchain_core.prompts import ChatPromptTemplate

    prompt = ChatPromptTemplate.from_messages([("system", SYSTEM), ("human", "Dispute facts (JSON):\n{facts}")])
    chain = prompt | init_chat_model(model_name, temperature=0).with_structured_output(Analysis)
    result: Analysis = chain.invoke({"facts": json.dumps(d.model_dump(), indent=2)})
    result.engine = model_name
    return result


@app.get("/healthz")
def healthz():
    return {"ok": True, "engine": os.getenv("DD_MODEL") or "heuristic"}


@app.post("/analyze", response_model=Analysis)
def analyze(d: DisputeIn):
    model = os.getenv("GEMINI_API")
    if not model:
        return heuristic(d)
    try:
        return langchain_analyze(d, model)
    except Exception as e:  # surface model/config errors to the Go backend as a 502-able error
        raise HTTPException(status_code=502, detail=f"LLM call failed: {e}")
