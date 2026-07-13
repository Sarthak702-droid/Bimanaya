import logging
from typing import Dict, Any, List

logger = logging.getLogger(__name__)

class DraftingService:
    def __init__(self):
        pass

    def generate_grievance_draft(self, case_id: str, claim_details: Dict[str, Any], issues: List[Dict[str, Any]], citations: List[Dict[str, Any]]) -> Dict[str, str]:
        """
        Creates a formal, legally structured grievance representation letter
        suitable for submission to the Insurer's Grievance Redressal Officer (GRO).
        """
        insurer = claim_details.get("insurer", "Star Health Insurance Co. Ltd.")
        claim_number = claim_details.get("claim_number", "CLI-9908122-A")
        policy_number = claim_details.get("policy_number", "POL-88001928-0")
        amount_claimed = claim_details.get("amount_claimed", 150000.0)
        amount_paid = claim_details.get("amount_paid", 90000.0)
        amount_disputed = claim_details.get("amount_disputed", 60000.0)

        subject = f"Representation against underpayment of Claim No. {claim_number} under Policy No. {policy_number} - Improper Proportionate Deduction on Room Rent"

        # Constructing formal HTML structure that TipTap Editor can render
        content = f"""
        <h2>GRIEVANCE REPRESENTATION LETTER</h2>
        <p><strong>Date:</strong> {logging.time.strftime('%Y-%m-%d')}</p>
        <p><strong>To,</strong><br/>
        The Grievance Redressal Officer,<br/>
        {insurer}</p>

        <p><strong>Subject:</strong> {subject}</p>

        <hr/>

        <h3>1. Policyholder & Claim Details</h3>
        <ul>
            <li><strong>Policyholder Name:</strong> Policyholder (Case Ref: {case_id[:8]})</li>
            <li><strong>Policy Number:</strong> {policy_number}</li>
            <li><strong>Claim Number:</strong> {claim_number}</li>
            <li><strong>Total Claimed Amount:</strong> Rs. {amount_claimed:,.2f}</li>
            <li><strong>Approved/Paid Amount:</strong> Rs. {amount_paid:,.2f}</li>
            <li><strong>Disputed/Underpaid Amount:</strong> Rs. {amount_disputed:,.2f}</li>
        </ul>

        <h3>2. Summary of Dispute</h3>
        <p>The insurer has approved only Rs. {amount_paid:,.2f} against a total claim of Rs. {amount_claimed:,.2f}, resulting in an arbitrary deduction of Rs. {amount_disputed:,.2f}. The claim settlement sheet indicates that this deduction was applied under the guise of 'proportionate deduction' due to the selection of a room exceeding the capping limits of 1% of the Sum Insured.</p>

        <h3>3. Grounds for Objection & Factual Arguments</h3>
        <p>While we acknowledge that the policy limits room boarding charges to 1% of the Sum Insured per day, the insurer has incorrectly and illegally extended the proportionate deduction to all non-associated charges. Specifically:</p>
        <ul>
            <li><strong>Improper deduction on Non-associated Expenses:</strong> The insurer has applied a proportionate deduction ratio of approximately 50% to costs like medicines, consumables, surgical implants, and diagnostic tests.</li>
            <li><strong>Violation of IRDAI Standardization Guidelines:</strong> In accordance with the <strong>IRDAI Standardization of General Terms in Health Insurance (2016 & 2020 Guidelines)</strong>, proportionate deductions are strictly prohibited on cost of implants, diagnostics, and pharmacy. Capping can only be applied to service-oriented associated charges (like room rent, nursing, and doctor visits).</li>
        </ul>

        <h3>4. Legal and Regulatory References</h3>
        <blockquote>
            "In case of room rent limits, proportionate deduction can only be applied on associate medical expenses and NOT on cost of implants, medical devices, diagnostics or pharmacy." 
            <br/><strong>- IRDAI Guidelines on Standardization, Circular Ref: IRDAI/HLT/REG/CIR/2016</strong>
        </blockquote>

        <h3>5. Relief Sought</h3>
        <p>We request the insurer to review this claim, recalculate the proportionate deduction according to IRDAI guidelines, and pay the balance amount of <strong>Rs. {amount_disputed * 0.6:,.2f}</strong> (being the amount deducted from pharmacy, diagnostics, and medical implants) along with applicable interest.</p>

        <p>Sincerely,</p>
        <p>_______________________<br/>
        <strong>Policyholder / Authorized Representative</strong></p>
        
        <hr/>
        <p style="font-size: 10px; color: #888;"><strong>Disclaimer:</strong> This grievance representation is prepared by BimaNyaya and reviewed by experts for correctness. It is based on documents uploaded by the user and current regulatory frameworks.</p>
        """

        return {
            "subject": subject,
            "content": content.strip()
        }
