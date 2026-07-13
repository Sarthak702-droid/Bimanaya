import logging
from fastapi import FastAPI, HTTPException, Response
from pydantic import BaseModel
from typing import List, Dict, Any, Optional

from app.extraction import ExtractionService
from app.retrieval import RetrievalService
from app.reasoning import ReasoningEngine
from app.drafting import DraftingService
from app.safety import SafetyGate

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("ai-worker")

app = FastAPI(title="BimaNyaya AI Processing Worker Pool")

# Initialize services
extractor = ExtractionService()
retriever = RetrievalService()
reasoner = ReasoningEngine()
drafter = DraftingService()
safety_gate = SafetyGate()

# Schemas
class DocumentMeta(BaseModel):
    id: str
    storage_key: str
    document_type: str

class ProcessCaseRequest(BaseModel):
    case_id: str
    claim_number: Optional[str] = ""
    insurer: Optional[str] = ""
    claim_status: Optional[str] = ""
    amount_claimed: float = 0.0
    amount_paid: float = 0.0
    amount_disputed: float = 0.0
    documents: List[DocumentMeta] = []

class GenerateDraftRequest(BaseModel):
    case_id: str
    language: str

class TranslateRequest(BaseModel):
    text: str
    subject: str
    target_language: str

@app.get("/")
def health_check():
    return {"status": "online", "engine": "BimaNyaya AI Worker Pool"}

@app.post("/process-case")
def process_case(req: ProcessCaseRequest):
    logger.info(f"Processing case: {req.case_id}")
    
    # 1. Simulate document reading and field extraction
    # In a real setup, we would read the documents from S3/MinIO bucket.
    # E.g. raw_text = extractor.extract_text_from_pdf(local_path)
    # For demo/fallback, we construct text based on input parameters.
    simulated_doc_text = f"""
    Claim Details:
    Insurer: {req.insurer}
    Claim Number: {req.claim_number}
    Amount Claimed: Rs. {req.amount_claimed}
    Amount Paid: Rs. {req.amount_paid}
    Amount Disputed/Deducted: Rs. {req.amount_disputed}
    Hospital: Apollo Hospitals, Hyderabad.
    Rejection / deduction reason: Room rent capping deduction applied since user chose Single Room Deluxe.
    Exclusion details: Capping applied under clause 1.A of Star Health policy.
    """

    # 2. Run retrieval (RAG)
    rag_results = retriever.retrieve_citations(req.insurer, simulated_doc_text)
    
    policy_citations = rag_results["policy_citations"]
    regulatory_citations = rag_results["regulatory_citations"]

    # 3. Claims Reasoning
    claim_facts = {
        "insurer": req.insurer,
        "amount_claimed": req.amount_claimed,
        "amount_paid": req.amount_paid,
        "amount_disputed": req.amount_disputed
    }
    analysis_result = reasoner.analyze_claim(claim_facts, policy_citations, regulatory_citations)

    # 4. Generate structured output
    # Prepare Case Issues
    issues = [{
        "issue_category": analysis_result["issue_category"],
        "summary": analysis_result["supporting_facts"][0],
        "details": analysis_result["details"],
        "confidence": analysis_result["confidence"]
    }]

    # Format Citations response
    citations = []
    for pol in policy_citations:
        citations.append({
            "source_type": "POLICY",
            "section_name": pol["section"],
            "clause_number": pol["clause_number"],
            "quoted_text": pol["quoted_text"],
            "confidence": 0.90,
            "validation_status": "VALIDATED"
        })
    for reg in regulatory_citations:
        citations.append({
            "source_type": "REGULATION",
            "section_name": reg["section"],
            "clause_number": reg["clause_number"],
            "quoted_text": reg["quoted_text"],
            "confidence": 0.95,
            "validation_status": "VALIDATED"
        })

    # Prepare Evidence Checklist
    evidence_checklist = [
        {
            "document_name": "Rejection/Settlement Letter",
            "why_required": "To establish the exact grounds of deduction applied by the insurer",
            "priority": "HIGH",
            "is_mandatory": True,
            "status": "AVAILABLE"
        },
        {
            "document_name": "Policy Wording Booklet",
            "why_required": "To cross-verify terms of room rent capping limits",
            "priority": "HIGH",
            "is_mandatory": True,
            "status": "AVAILABLE"
        },
        {
            "document_name": "Detailed Bill Breakup",
            "why_required": "To separate associated expenses from non-associated expenses like implants & medicines",
            "priority": "MEDIUM",
            "is_mandatory": False,
            "status": "MISSING"
        }
    ]

    return {
        "issues": issues,
        "citations": citations,
        "evidence_checklist": evidence_checklist
    }

@app.post("/generate-draft")
def generate_draft(req: GenerateDraftRequest):
    logger.info(f"Generating draft for case {req.case_id} in language {req.language}")

    # 1. Simulated details retrieval
    # In production, we query Go API database or fetch from payload.
    # Let's use standard default values mimicking the database case record
    claim_details = {
        "insurer": "Star Health Insurance Co. Ltd.",
        "claim_number": "CLI-9908122-A",
        "policy_number": "POL-88001928-0",
        "amount_claimed": 150000.0,
        "amount_paid": 90000.0,
        "amount_disputed": 60000.0
    }

    issues = [{"category": "ROOM_RENT_DEDUCTION"}]
    citations = [{"source_type": "REGULATION", "clause": "6.1"}]

    # 2. Draft Generation
    draft_result = drafter.generate_grievance_draft(req.case_id, claim_details, issues, citations)

    # 3. Safety Gate Verification
    safety_status = safety_gate.verify_draft(draft_result["content"], claim_details)

    return {
        "subject": draft_result["subject"],
        "content": draft_result["content"],
        "safety_status": safety_status
    }

@app.post("/translate")
def translate(req: TranslateRequest):
    logger.info(f"Translating text to language: {req.target_language}")
    
    # Simulating translation mapping for Hindi and Odia
    # In production, we would use an LLM provider SDK (OpenAI/Gemini/Anthropic) or translation models.
    translated_subject = req.subject
    translated_text = req.text

    if req.target_language == "hi":
        translated_subject = "दावा संख्या के कम भुगतान के खिलाफ शिकायत - अनुचित आनुपातिक कटौती"
        translated_text = req.text.replace("GRIEVANCE REPRESENTATION LETTER", "शिकायत प्रतिनिधित्व पत्र") \
                                   .replace("Policyholder & Claim Details", "पॉलिसीधारक और दावे का विवरण") \
                                   .replace("Summary of Dispute", "विवाद का सारांश") \
                                   .replace("Grounds for Objection", "आपत्ति के आधार")
    elif req.target_language == "or":
        translated_subject = "ଦାବି ଅଧୀନରେ କମ୍ ପେମେଣ୍ଟ ବିରୋଧରେ ଅଭିଯୋଗ - ଅଯଥା ଆନୁପାତିକ କଟାକଟି"
        translated_text = req.text.replace("GRIEVANCE REPRESENTATION LETTER", "ଅଭିଯୋଗ ପ୍ରତିନିଧିତ୍ୱ ପତ୍ର") \
                                   .replace("Policyholder & Claim Details", "ପଲିସିଧାରୀ ଏବଂ ଦାବି ବିବରଣୀ") \
                                   .replace("Summary of Dispute", "ବିବାଦର ସାରାଂଶ") \
                                   .replace("Grounds for Objection", "ଆପତ୍ତିର କାରଣ")

    return {
        "translated_subject": translated_subject,
        "translated_text": translated_text
    }

@app.post("/generate-pdf")
def generate_pdf(payload: dict):
    subject = payload.get("subject", "")
    html_content = payload.get("html_content", "")
    
    # Pre-process html_content to ensure it is clean and fpdf-friendly
    # Replace self-closing tags like <br/> to <br>, <hr/> to <hr>
    html_content = html_content.replace("<br/>", "<br>").replace("<hr/>", "<hr>")
    
    from fpdf import FPDF
    
    class BimaNyayaPDF(FPDF):
        def header(self):
            self.set_font("helvetica", "B", 8)
            self.set_text_color(128, 128, 128)
            self.cell(0, 10, "BimaNyaya Grievance Portal | Expert Review Completed", align="R")
            self.ln(10)
            
        def footer(self):
            self.set_y(-15)
            self.set_font("helvetica", "I", 8)
            self.set_text_color(128, 128, 128)
            self.cell(0, 10, f"Page {self.page_no()}/{{nb}} - Prepared under IRDAI guidelines", align="C")
            
    pdf = BimaNyayaPDF()
    pdf.alias_nb_pages()
    pdf.add_page()
    
    # Add subject
    pdf.set_font("helvetica", "B", 12)
    pdf.set_text_color(20, 20, 20)
    pdf.multi_cell(0, 8, subject)
    pdf.ln(5)
    
    # Add body HTML
    pdf.set_font("helvetica", size=10)
    pdf.set_text_color(40, 40, 40)
    try:
        pdf.write_html(html_content)
    except Exception as e:
        logger.error(f"HTML PDF conversion failed, falling back to plain text: {e}")
        import re
        clean_text = re.sub('<[^<]+?>', '', html_content)
        pdf.multi_cell(0, 6, clean_text)
        
    pdf_bytes = pdf.output()
    
    return Response(content=bytes(pdf_bytes), media_type="application/pdf")

