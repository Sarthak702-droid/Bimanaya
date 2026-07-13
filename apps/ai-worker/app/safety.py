import re
import logging
from typing import Dict, Any, List

logger = logging.getLogger(__name__)

class SafetyGate:
    def __init__(self):
        pass

    def verify_draft(self, draft_content: str, claim_details: Dict[str, Any]) -> str:
        """
        Scans a draft and returns safety status: PASS, WARNING, BLOCK
        """
        content_lower = draft_content.lower()

        # 1. Check for aggressive/defamatory accusations (BLOCK)
        block_words = ["fraudulent", "cheating", "cheat", "scam", "thief", "steal", "con artist", "extortion"]
        for word in block_words:
            if word in content_lower:
                logger.warning(f"Safety Gate blocked draft due to word: {word}")
                return "BLOCK"

        # 2. Check for over-guarantee language (WARNING)
        warning_phrases = ["guarantee 100% payment", "must refund immediately", "will definitely win", "guarantee a win"]
        for phrase in warning_phrases:
            if phrase in content_lower:
                logger.warning(f"Safety Gate flagged draft warning due to phrase: {phrase}")
                return "WARNING"

        # 3. Check for amount integrity mismatches
        # Extract numbers from draft and match with claim details
        amount_claimed = claim_details.get("amount_claimed", 0.0)
        # Check if amount_claimed is present in content string (ignoring format)
        amount_str = f"{amount_claimed:,.0f}".replace(",", "") # e.g. 150000
        if amount_str not in content_lower.replace(",", ""):
            # If the exact amount claimed isn't found in text, raise a warning
            logger.warning(f"Claimed amount {amount_str} not matching text values.")
            return "WARNING"

        # 4. Check for PII Leakages (e.g. Aadhaar Card formats, PAN formats)
        # Aadhaar: 12 digits, PAN: 5 letters, 4 digits, 1 letter
        aadhaar_pattern = r"\b\d{4}\s\d{4}\s\d{4}\b"
        pan_pattern = r"\b[a-zA-Z]{5}\d{4}[a-zA-Z]\b"

        if re.search(aadhaar_pattern, draft_content) or re.search(pan_pattern, draft_content):
            logger.warning("Safety Gate blocked draft due to PII leak (Aadhaar or PAN pattern found)")
            return "BLOCK"

        return "PASS"
