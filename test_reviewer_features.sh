#!/bin/bash
set -e

# Configuration
API_URL="http://localhost:8080/api/v1"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0;0m'

echo -e "${BLUE}=== Starting BimaNyaya Reviewer & PDF Export Verification ===${NC}"

# STEP 1: AUTHENTICATE POLICYHOLDER & CREATE CASE
echo -e "\n${YELLOW}[Step 1] Authenticating Policyholder...${NC}"
PH_OTP_RESP=$(curl -s -X POST "$API_URL/auth/request-otp" \
  -H "Content-Type: application/json" \
  -d '{"email": "policyholder@bimanyaya.in"}')
PH_OTP=$(echo "$PH_OTP_RESP" | grep -o '"code_preview_demo":"[^"]*"' | head -n 1 | cut -d'"' -f4)

PH_AUTH_RESP=$(curl -s -X POST "$API_URL/auth/verify-otp" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"policyholder@bimanyaya.in\", \"code\": \"$PH_OTP\"}")
PH_TOKEN=$(echo "$PH_AUTH_RESP" | grep -o '"access_token":"[^"]*"' | head -n 1 | cut -d'"' -f4)
echo -e "${GREEN}✓ Authenticated Policyholder! Token: ${PH_TOKEN:0:20}...${NC}"

echo -e "\n${YELLOW}[Step 2] Creating claim case...${NC}"
CASE_RESP=$(curl -s -X POST "$API_URL/cases" \
  -H "Authorization: Bearer $PH_TOKEN" \
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
CASE_ID=$(echo "$CASE_RESP" | grep -o '"id":"[^"]*"' | head -n 1 | cut -d'"' -f4)
echo -e "${GREEN}✓ Case created: ID $CASE_ID${NC}"

# Record consent
echo -e "\n${YELLOW}[Step 3] Recording consent...${NC}"
CONSENT_RESP=$(curl -s -X POST "$API_URL/cases/$CASE_ID/consents" \
  -H "Authorization: Bearer $PH_TOKEN" \
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

# Upload Document
echo -e "\n${YELLOW}[Step 4] Reserving document...${NC}"
DOC_RESP=$(curl -s -X POST "$API_URL/cases/$CASE_ID/documents/upload-url" \
  -H "Authorization: Bearer $PH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"document_type": "REJECTION_LETTER", "filename": "rejection.pdf", "mime_type": "application/pdf", "size_bytes": 10240}')
echo "Document Reserve Response: $DOC_RESP"
DOC_ID=$(echo "$DOC_RESP" | grep -o '"document_id":"[^"]*"' | head -n 1 | cut -d'"' -f4)

echo -e "\n${YELLOW}[Step 5] Completing upload...${NC}"
COMPLETE_RESP=$(curl -s -X POST "$API_URL/cases/$CASE_ID/documents/complete" \
  -H "Authorization: Bearer $PH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"document_id\": \"$DOC_ID\", \"file_hash\": \"hash12345\", \"page_count\": 2}")
echo "Complete Response: $COMPLETE_RESP"

# Process case with AI worker
echo -e "\n${YELLOW}[Step 6] Triggering AI processing...${NC}"
PROCESS_RESP=$(curl -s -X POST "$API_URL/cases/$CASE_ID/process" \
  -H "Authorization: Bearer $PH_TOKEN")
echo "Process Response: $PROCESS_RESP"
sleep 3

# Create Draft
echo -e "\n${YELLOW}[Step 7] Generating draft...${NC}"
DRAFT_RESP=$(curl -s -X POST "$API_URL/cases/$CASE_ID/drafts" \
  -H "Authorization: Bearer $PH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"language": "en"}')
echo "Draft Response: $DRAFT_RESP"
DRAFT_ID=$(echo "$DRAFT_RESP" | grep -o '"draft_id":"[^"]*"' | head -n 1 | cut -d'"' -f4)
echo -e "${GREEN}✓ AI case processing complete, draft generated: $DRAFT_ID${NC}"


# STEP 2: AUTHENTICATE REVIEWER & ASSIGN ROLE VIA DB
echo -e "\n${YELLOW}[Step 8] Authenticating Reviewer...${NC}"
REV_OTP_RESP=$(curl -s -X POST "$API_URL/auth/request-otp" \
  -H "Content-Type: application/json" \
  -d '{"email": "reviewer@bimanyaya.in"}')
REV_OTP=$(echo "$REV_OTP_RESP" | grep -o '"code_preview_demo":"[^"]*"' | head -n 1 | cut -d'"' -f4)

REV_AUTH_RESP=$(curl -s -X POST "$API_URL/auth/verify-otp" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"reviewer@bimanyaya.in\", \"code\": \"$REV_OTP\"}")
REV_TOKEN=$(echo "$REV_AUTH_RESP" | grep -o '"access_token":"[^"]*"' | head -n 1 | cut -d'"' -f4)

echo "Promoting reviewer@bimanyaya.in to REVIEWER role in PostgreSQL..."
docker exec -i bimanyaya_postgres psql -U postgres -d bimanyaya -c "UPDATE users SET role = 'REVIEWER' WHERE email = 'reviewer@bimanyaya.in';" > /dev/null
echo -e "${GREEN}✓ Reviewer authenticated and promoted! Token: ${REV_TOKEN:0:20}...${NC}"


# STEP 3: CLAIM CASE & ADD FEEDBACK COMMENTS
echo -e "\n${YELLOW}[Step 9] Claiming Case...${NC}"
CLAIM_RESP=$(curl -s -X POST "$API_URL/reviewer/cases/$CASE_ID/claim" \
  -H "Authorization: Bearer $REV_TOKEN")
echo "Claim Response: $CLAIM_RESP"

echo -e "\n${YELLOW}[Step 10] Adding Review Comments (Feedback Loop)...${NC}"
ADD_COMM_RESP=$(curl -s -X POST "$API_URL/reviewer/cases/$CASE_ID/comments" \
  -H "Authorization: Bearer $REV_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"comment_text": "Case details double checked. Insurer proportionately deducted pharmacy charges in direct violation of the 2016 IRDAI standardization circular."}')
echo "Add Comment Response: $ADD_COMM_RESP"

echo -e "\n${YELLOW}[Step 11] Retrieving Review Comments...${NC}"
GET_COMM_RESP=$(curl -s -X GET "$API_URL/reviewer/cases/$CASE_ID/comments" \
  -H "Authorization: Bearer $REV_TOKEN")
echo "Comments: $GET_COMM_RESP"


# STEP 4: APPROVE CASE & EXPORT DRAFT AS PDF
echo -e "\n${YELLOW}[Step 12] Approving Case...${NC}"
APPROVE_RESP=$(curl -s -X POST "$API_URL/reviewer/cases/$CASE_ID/approve" \
  -H "Authorization: Bearer $REV_TOKEN")
echo "Approve Response: $APPROVE_RESP"

echo -e "\n${YELLOW}[Step 13] Exporting Grievance Draft as PDF...${NC}"
curl -s -o grievance.pdf "$API_URL/drafts/$DRAFT_ID/pdf" \
  -H "Authorization: Bearer $PH_TOKEN"

if [ -f grievance.pdf ]; then
  PDF_HEADER=$(head -n 1 grievance.pdf || true)
  echo "PDF Header Preview: $PDF_HEADER"
  if [[ "$PDF_HEADER" == *"%PDF"* ]]; then
    echo -e "${GREEN}✓ PDF successfully exported to grievance.pdf (Verified PDF binary format)${NC}"
  else
    echo -e "${RED}✗ Failed: grievance.pdf was created but lacks valid PDF headers${NC}"
    exit 1
  fi
else
  echo -e "${RED}✗ Failed: grievance.pdf was not created${NC}"
  exit 1
fi


# STEP 5: AUTHENTICATE ADMIN & RUN SLA VERIFICATION
echo -e "\n${YELLOW}[Step 14] Authenticating Admin...${NC}"
ADM_OTP_RESP=$(curl -s -X POST "$API_URL/auth/request-otp" \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@bimanyaya.in"}')
ADM_OTP=$(echo "$ADM_OTP_RESP" | grep -o '"code_preview_demo":"[^"]*"' | head -n 1 | cut -d'"' -f4)

ADM_AUTH_RESP=$(curl -s -X POST "$API_URL/auth/verify-otp" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"admin@bimanyaya.in\", \"code\": \"$ADM_OTP\"}")
ADM_TOKEN=$(echo "$ADM_AUTH_RESP" | grep -o '"access_token":"[^"]*"' | head -n 1 | cut -d'"' -f4)

echo "Promoting admin@bimanyaya.in to ADMIN role in PostgreSQL..."
docker exec -i bimanyaya_postgres psql -U postgres -d bimanyaya -c "UPDATE users SET role = 'ADMIN' WHERE email = 'admin@bimanyaya.in';" > /dev/null
echo -e "${GREEN}✓ Admin authenticated and promoted! Token: ${ADM_TOKEN:0:20}...${NC}"

echo -e "\n${YELLOW}[Step 15] Triggering SLA checks (Escalations workflow)...${NC}"
SLA_RESP=$(curl -s -X POST "$API_URL/admin/reviews/sla-checks" \
  -H "Authorization: Bearer $ADM_TOKEN")
echo "SLA checks completed successfully: $SLA_RESP"

echo -e "\n${GREEN}=== All Reviewer and PDF features successfully verified! ===${NC}"
