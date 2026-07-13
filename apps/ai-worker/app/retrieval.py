import os
import logging
from typing import List, Dict, Any

logger = logging.getLogger(__name__)

# Predefined Knowledge Corpus for Claims and Regulations (ICU and Room Rent limit issues)
IRDAI_REGULATIONS = [
    {
        "id": "reg_001",
        "authority": "IRDAI",
        "title": "Guidelines on Standardization of General Terms in Health Insurance Policies (2020)",
        "section": "Annexure-I: Standard Definitions",
        "clause_number": "12",
        "quoted_text": "Room Rent means the amount charged by a hospital for the occupancy of a bed on per day (24 hours) basis and shall include associated medical expenses.",
        "content": "Room Rent definition standardisation. Insurers must explicitly state room rent limits in the policy schedule. Associate medical expenses must be defined clearly, and proportionate deductions can only be applied to expenses directly linked to room category.",
        "effective_date": "2020-10-01"
    },
    {
        "id": "reg_002",
        "authority": "IRDAI",
        "title": "Circular on Proportionate Deduction on Room Rent limits (IRDAI/HLT/REG/CIR/2016)",
        "section": "Standardization Guidelines - Section 6",
        "clause_number": "6.1",
        "quoted_text": "In case of room rent limits, proportionate deduction can only be applied on associate medical expenses (e.g. nursing, doctors fees, operation theatre charges) and NOT on cost of implants, medical devices, diagnostics or pharmacy.",
        "content": "Proportionate deductions rules. Insurers are prohibited from making proportionate deductions on pharmacy, medical devices, implants, and diagnostics. They can only apply it on service-oriented items like nursing, consulting, and OT charges if room rent exceeds the eligible limit.",
        "effective_date": "2016-07-29"
    }
]

POLICY_TEMPLATES = [
    {
        "id": "pol_001",
        "insurer_name": "Star Health",
        "product_name": "Family Health Optima",
        "section": "Section 1 - Benefits",
        "clause_number": "1.A",
        "quoted_text": "Room, Boarding and Nursing Expenses as provided by the Hospital / Nursing Home at 1% of the Sum Insured per day subject to a maximum of Rs. 5,000/- per day.",
        "content": "Room rent limits: 1% of Sum Insured per day. ICU limits: 2% of Sum Insured per day. If a higher room category is chosen, associated medical expenses will be proportionately reduced.",
        "version": "v1.2",
        "effective_date": "2021-04-01"
    },
    {
        "id": "pol_002",
        "insurer_name": "Niva Bupa",
        "product_name": "ReAssure",
        "section": "Section 2 - Inpatient Care",
        "clause_number": "2.3",
        "quoted_text": "No room rent capping. Up to Single Private Suite Room eligibility depending on Sum Insured options.",
        "content": "ReAssure policy has no capping on room rent if Single Private Room is opted, except for specified list of sub-limits.",
        "version": "v2.0",
        "effective_date": "2022-01-15"
    }
]

class RetrievalService:
    def __init__(self):
        # In a production environment with vector DB, we initialize sentence-transformers
        # E.g. self.model = SentenceTransformer('all-MiniLM-L6-v2')
        pass

    def retrieve_citations(self, insurer: str, query: str) -> Dict[str, List[Dict[str, Any]]]:
        """
        Retrieves matching policy clauses and IRDAI regulations
        """
        matched_policies = []
        matched_regulations = []

        query_lower = query.lower()
        insurer_lower = insurer.lower() if insurer else ""

        # Simple semantic/keyword matching fallback
        # 1. Match Policy Templates
        for pol in POLICY_TEMPLATES:
            # Match insurer name
            if insurer_lower and insurer_lower in pol["insurer_name"].lower():
                matched_policies.append(pol)
            # Match keywords
            elif "room" in query_lower or "limit" in query_lower:
                if pol not in matched_policies:
                    matched_policies.append(pol)

        # 2. Match IRDAI Regulations
        for reg in IRDAI_REGULATIONS:
            if "deduction" in query_lower or "proportionate" in query_lower or "limit" in query_lower or "room" in query_lower:
                matched_regulations.append(reg)

        return {
            "policy_citations": matched_policies,
            "regulatory_citations": matched_regulations
        }
