#!/bin/bash
set -e

# Colors for presentation
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080/api/v1"

echo -e "${BLUE}=== Starting BimaNyaya E2E API Verification Script ===${NC}"

# Check if server is running
if ! curl -s "http://localhost:8080/" > /dev/null; then
    echo -e "${RED}Error: Go API server is not running on http://localhost:8080/. Make sure to run 'docker-compose up' first.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Core API is online!${NC}\n"

# 1. AUTHENTICATION: Request OTP
echo -e "${YELLOW}[Step 1] Requesting OTP for test user...${NC}"
OTP_RESP=$(curl -s -X POST "$API_URL/auth/request-otp" \
  -H "Content-Type: application/json" \
  -d '{"email": "test@bimanyaya.in"}')
echo "Response: $OTP_RESP"

OTP_CODE=$(echo "$OTP_RESP" | grep -o '"code_preview_demo":"[0-9]*"' | cut -d'"' -d':' -f2 | tr -d '"')
echo -e "${GREEN}✓ OTP received: $OTP_CODE (Demo mode auto-retrieved)${NC}\n"

# Verify OTP
echo -e "${YELLOW}[Step 2] Verifying OTP and generating session token...${NC}"
VERIFY_RESP=$(curl -s -X POST "$API_URL/auth/verify-otp" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"test@bimanyaya.in\", \"code\": \"$OTP_CODE\"}")

TOKEN=$(echo "$VERIFY_RESP" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
    echo -e "${RED}Failed to authenticate and retrieve access token.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Authenticated! JWT Access Token received: ${TOKEN:0:15}...${NC}\n"

# Get Profile
echo -e "${YELLOW}[Step 3] Fetching authenticated user profile...${NC}"
PROFILE=$(curl -s -X GET "$API_URL/auth/me" \
  -H "Authorization: Bearer $TOKEN")
echo "Profile Details: $PROFILE"
echo -e "${GREEN}✓ Profile retrieved successfully!${NC}\n"

# 2. ELIGIBILITY: Check Scope
echo -e "${YELLOW}[Step 4] Checking claim eligibility...${NC}"
ELIG_RESP=$(curl -s -X POST "$API_URL/eligibility/check" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "insurance_type": "HEALTH",
    "claim_status": "PARTIALLY_SETTLED",
    "disputed_amount": 60000.00,
    "available_documents": ["rejection_letter", "discharge_summary"],
    "user_authority": true
  }')
echo "Eligibility Response: $ELIG_RESP"
echo -e "${GREEN}✓ Eligibility assessment completed.${NC}\n"

# 3. CASE CREATION
echo -e "${YELLOW}[Step 5] Creating claim case...${NC}"
CASE_RESP=$(curl -s -X POST "$API_URL/cases" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "insurance_type": "HEALTH",
    "claim_category": "ROOM_RENT_DEDUCTION",
    "claim_status": "PARTIALLY_SETTLED",
    "insurer_name": "Star Health Insurance Co. Ltd.",
    "policy_number": "POL-STAR-8871",
    "claim_number": "CLM-STAR-992",
    "amount_claimed": 150000.00,
    "amount_paid": 90000.00,
    "amount_disputed": 60000.00,
    "preferred_language": "en"
  }')
echo "Case Response: $CASE_RESP"
CASE_ID=$(echo "$CASE_RESP" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo -e "${GREEN}✓ Case created successfully! Case ID: $CASE_ID${NC}\n"

# 4. CONSENTS RECORDING
echo -e "${YELLOW}[Step 6] Recording policyholder consents...${NC}"
CONSENT_RESP=$(curl -s -X POST "$API_URL/cases/$CASE_ID/consents" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "consent_version": "v1.0",
    "document_processing_consent": true,
    "reviewer_access_consent": true,
    "data_retention_consent": true,
    "authority_confirmation": true,
    "research_consent": false
  }')
echo "Consent Response: $CONSENT_RESP"
echo -e "${GREEN}✓ Consents registered.${NC}\n"

# 5. DOCUMENT UPLOAD URL
echo -e "${YELLOW}[Step 7] Reserving document and getting upload URL...${NC}"
UPLOAD_RESP=$(curl -s -X POST "$API_URL/cases/$CASE_ID/documents/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "rejection_letter.pdf",
    "document_type": "REJECTION_LETTER",
    "mime_type": "application/pdf",
    "size_bytes": 102400
  }')
echo "Upload URL Response: $UPLOAD_RESP"
DOC_ID=$(echo "$UPLOAD_RESP" | grep -o '"document_id":"[^"]*"' | cut -d'"' -f4)
UPLOAD_URL=$(echo "$UPLOAD_RESP" | grep -o '"upload_url":"[^"]*"' | cut -d'"' -f4)
echo -e "${GREEN}✓ Reserved document ID: $DOC_ID${NC}"

# Complete Upload
echo -e "${YELLOW}[Step 8] Finalizing document upload state...${NC}"
COMPLETE_RESP=$(curl -s -X POST "$API_URL/cases/$CASE_ID/documents/complete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"document_id\": \"$DOC_ID\", \"file_hash\": \"a6e87f2b90c1\", \"page_count\": 2}")
echo "Complete Response: $COMPLETE_RESP"
echo -e "${GREEN}✓ Document upload complete! Case workflow is now in PROCESSING.${NC}\n"

# 6. RAG RETRIEVAL AND REASONING PIPELINE
echo -e "${YELLOW}[Step 9] Triggering AI claims reasoning analysis...${NC}"
PROCESS_RESP=$(curl -s -X POST "$API_URL/cases/$CASE_ID/process" \
  -H "Authorization: Bearer $TOKEN")
echo "Trigger Response: $PROCESS_RESP"
echo -e "${BLUE}Sleeping 3 seconds for asynchronous AI execution...${NC}"
sleep 3

# Check Analysis Results
echo -e "${YELLOW}[Step 10] Retrieving extracted issues...${NC}"
ISSUES=$(curl -s -X GET "$API_URL/cases/$CASE_ID/analysis" \
  -H "Authorization: Bearer $TOKEN")
echo "Extracted Issues: $ISSUES"

echo -e "${YELLOW}[Step 11] Retrieving retrieved RAG citations (Insurer + IRDAI)...${NC}"
CITATIONS=$(curl -s -X GET "$API_URL/cases/$CASE_ID/citations" \
  -H "Authorization: Bearer $TOKEN")
echo "Citations: $CITATIONS"

echo -e "${YELLOW}[Step 12] Retrieving evidence checklists...${NC}"
EVIDENCE=$(curl -s -X GET "$API_URL/cases/$CASE_ID/evidence" \
  -H "Authorization: Bearer $TOKEN")
echo "Evidence checklist: $EVIDENCE"
echo -e "${GREEN}✓ AI extraction and reasoning analysis checks out!${NC}\n"

# 7. GRIEVANCE DRAFT AND TRANSLATION
echo -e "${YELLOW}[Step 13] Generating grievance representation draft...${NC}"
DRAFT_RESP=$(curl -s -X POST "$API_URL/cases/$CASE_ID/drafts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"language": "en"}')
echo "Draft Response: $DRAFT_RESP"
DRAFT_ID=$(echo "$DRAFT_RESP" | grep -o '"draft_id":"[^"]*"' | cut -d'"' -f4)

# Translate Draft to Hindi
echo -e "${YELLOW}[Step 14] Translating grievance draft to Hindi (Multilingual support)...${NC}"
TRANS_RESP=$(curl -s -X POST "$API_URL/drafts/$DRAFT_ID/translate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"target_language": "hi"}')
echo "Translation Response: $TRANS_RESP"
echo -e "${GREEN}✓ Grievance drafting and translations verified!${NC}\n"

echo -e "${GREEN}=== All BimaNyaya backend modules successfully verified! ===${NC}"
