package eligibility

import (
	"encoding/json"
	"net/http"

	"bimanyaya/api/internal/db"
)

type CheckRequest struct {
	InsuranceType      string   `json:"insurance_type"`       // e.g. "HEALTH"
	ClaimStatus        string   `json:"claim_status"`         // e.g. "REJECTED", "PARTIALLY_SETTLED"
	DisputedAmount     float64  `json:"disputed_amount"`      // e.g. 50000.00
	AvailableDocuments []string `json:"available_documents"` // e.g. ["rejection_letter", "discharge_summary"]
	UserAuthority      bool     `json:"user_authority"`       // e.g. true (policyholder or authorized agent)
}

type CheckResponse struct {
	Status               string   `json:"status"` // ELIGIBLE, CONDITIONALLY_ELIGIBLE, MANUAL_REVIEW_REQUIRED, NOT_SUPPORTED
	MissingDocuments     []string `json:"missing_documents"`
	ManualReviewRequired bool     `json:"manual_review_required"`
	ReasonCodes          []string `json:"reason_codes"`
}

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

func (s *Service) CheckEligibility(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	var resp CheckResponse
	resp.MissingDocuments = make([]string, 0)
	resp.ReasonCodes = make([]string, 0)

	// Rule 1: Only HEALTH insurance is currently supported in this version
	if req.InsuranceType != "HEALTH" {
		resp.Status = "NOT_SUPPORTED"
		resp.ReasonCodes = append(resp.ReasonCodes, "UNSUPPORTED_INSURANCE_TYPE")
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Rule 2: Must be a rejected, partially settled claim, or denied cashless claim
	validStatus := false
	supportedStatuses := []string{"REJECTED", "PARTIALLY_SETTLED", "CASHLESS_DENIED", "PARTIALLY_PAID"}
	for _, status := range supportedStatuses {
		if req.ClaimStatus == status {
			validStatus = true
			break
		}
	}
	if !validStatus {
		resp.Status = "NOT_SUPPORTED"
		resp.ReasonCodes = append(resp.ReasonCodes, "UNSUPPORTED_CLAIM_STATUS")
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Rule 3: Legal authority check
	if !req.UserAuthority {
		resp.Status = "NOT_SUPPORTED"
		resp.ReasonCodes = append(resp.ReasonCodes, "MISSING_LEGAL_AUTHORITY")
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Rule 4: Document availability checks
	hasRejectionLetter := false
	hasDischargeSummary := false
	hasPolicyWording := false

	for _, doc := range req.AvailableDocuments {
		switch doc {
		case "rejection_letter", "settlement_letter", "rejection":
			hasRejectionLetter = true
		case "discharge_summary", "discharge":
			hasDischargeSummary = true
		case "policy_wording", "policy_schedule":
			hasPolicyWording = true
		}
	}

	if !hasRejectionLetter {
		resp.MissingDocuments = append(resp.MissingDocuments, "rejection_letter")
		resp.ReasonCodes = append(resp.ReasonCodes, "MISSING_REJECTION_LETTER")
	}
	if !hasDischargeSummary {
		resp.MissingDocuments = append(resp.MissingDocuments, "discharge_summary")
		resp.ReasonCodes = append(resp.ReasonCodes, "MISSING_DISCHARGE_SUMMARY")
	}
	if !hasPolicyWording {
		resp.MissingDocuments = append(resp.MissingDocuments, "policy_wording")
		resp.ReasonCodes = append(resp.ReasonCodes, "MISSING_POLICY_WORDING")
	}

	// Determine final status
	if len(resp.MissingDocuments) > 0 {
		resp.Status = "CONDITIONALLY_ELIGIBLE"
		resp.ManualReviewRequired = false
	} else {
		resp.Status = "ELIGIBLE"
		resp.ManualReviewRequired = false
	}

	// High disputed amount rules (over 5 Lakhs INR) triggers review flag
	if req.DisputedAmount > 500000.00 {
		resp.ManualReviewRequired = true
		resp.ReasonCodes = append(resp.ReasonCodes, "HIGH_DISPUTE_AMOUNT_FLAG")
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
