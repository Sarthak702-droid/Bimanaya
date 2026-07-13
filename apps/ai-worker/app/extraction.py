import re
import fitz  # PyMuPDF
import pdfplumber
import logging
from typing import Dict, Any, List

logger = logging.getLogger(__name__)

class ExtractionService:
    def __init__(self):
        pass

    def extract_text_from_pdf(self, file_path: str) -> str:
        text = ""
        try:
            doc = fitz.open(file_path)
            for page in doc:
                text += page.get_text()
            doc.close()
        except Exception as e:
            logger.error(f"Failed fitz extraction for {file_path}: {e}")
            try:
                # Fallback to pdfplumber
                with pdfplumber.open(file_path) as pdf:
                    for page in pdf.pages:
                        extracted = page.extract_text()
                        if extracted:
                            text += extracted
            except Exception as e2:
                logger.error(f"Fallback pdfplumber extraction failed: {e2}")
        return text

    def classify_document(self, text: str, filename: str) -> str:
        text_lower = text.lower()
        filename_lower = filename.lower()

        if "rejection" in text_lower or "repudiat" in text_lower or "reject" in filename_lower:
            return "REJECTION_LETTER"
        elif "schedule" in text_lower or "policy schedule" in text_lower or "schedule" in filename_lower:
            return "POLICY_SCHEDULE"
        elif "discharge summary" in text_lower or "clinical summary" in text_lower or "discharge" in filename_lower:
            return "DISCHARGE_SUMMARY"
        elif "wording" in text_lower or "terms and conditions" in text_lower or "wording" in filename_lower:
            return "POLICY_WORDING"
        elif "settlement" in text_lower or "claim settlement" in text_lower:
            return "SETTLEMENT_LETTER"
        else:
            return "SUPPORTING_MEDICAL_REPORT"

    def extract_fields(self, text: str) -> Dict[str, Any]:
        """
        Extract claim numbers, dates, policy details and amounts using regex heuristics
        """
        extracted = {}

        # 1. Claim Number
        claim_patterns = [
            r"Claim\s*(?:No|Number|Ref|Reference)?[:\s\-]+([A-Z0-9\-/\\]+)",
            r"ClaimID[:\s\-]+([A-Z0-9\-/\\]+)"
        ]
        for pattern in claim_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                extracted["claim_number"] = match.group(1).strip()
                break

        # 2. Policy Number
        policy_patterns = [
            r"Policy\s*(?:No|Number|Ref|Reference)?[:\s\-]+([A-Z0-9\-/\\]+)",
            r"PolicyID[:\s\-]+([A-Z0-9\-/\\]+)"
        ]
        for pattern in policy_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                extracted["policy_number"] = match.group(1).strip()
                break

        # 3. Insurer Name
        insurer_patterns = [
            r"(Niva\s+Bupa|Star\s+Health|Care\s+Health|HDFC\s+Ergo|ICICI\s+Lombard|Aditya\s+Birla)",
            r"Insurance\s+Company\s+Ltd\.?"
        ]
        for pattern in insurer_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                extracted["insurer_name"] = match.group(1).strip()
                break

        # 4. Amounts
        claimed_pattern = r"(?:claimed|claim\s+amount|amount\s+claimed)[:\s\-]*Rs\.?\s*([\d,]+(?:\.\d{2})?)"
        paid_pattern = r"(?:paid|settled|approved\s+amount|settled\s+amount)[:\s\-]*Rs\.?\s*([\d,]+(?:\.\d{2})?)"
        disputed_pattern = r"(?:disputed|deducted|deduction\s+amount)[:\s\-]*Rs\.?\s*([\d,]+(?:\.\d{2})?)"

        claimed_match = re.search(claimed_pattern, text, re.IGNORECASE)
        if claimed_match:
            extracted["amount_claimed"] = self.clean_amount(claimed_match.group(1))

        paid_match = re.search(paid_pattern, text, re.IGNORECASE)
        if paid_match:
            extracted["amount_paid"] = self.clean_amount(paid_match.group(1))

        disputed_match = re.search(disputed_pattern, text, re.IGNORECASE)
        if disputed_match:
            extracted["amount_disputed"] = self.clean_amount(disputed_match.group(1))

        return extracted

    def clean_amount(self, amt_str: str) -> float:
        try:
            return float(amt_str.replace(",", "").strip())
        except ValueError:
            return 0.0
