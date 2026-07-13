CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- User Roles Enum or Type
CREATE TYPE user_status AS ENUM ('ACTIVE', 'SUSPENDED', 'PENDING_VERIFICATION');
CREATE TYPE workflow_state AS ENUM (
    'DRAFT',
    'ELIGIBILITY_COMPLETED',
    'CONSENT_PENDING',
    'DOCUMENTS_PENDING',
    'PROCESSING',
    'NEEDS_CLARIFICATION',
    'ANALYSIS_READY',
    'REVIEW_REQUIRED',
    'IN_REVIEW',
    'MORE_INFORMATION_REQUIRED',
    'APPROVED',
    'READY_FOR_EXPORT',
    'SUBMITTED',
    'TRACKING',
    'RESOLVED',
    'CLOSED',
    'DELETION_PENDING',
    'DELETED'
);
CREATE TYPE risk_level AS ENUM ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL');
CREATE TYPE document_status AS ENUM ('UPLOADED', 'SCANNING', 'PROCESSING', 'OCR_RUNNING', 'READY', 'NEEDS_BETTER_COPY', 'REJECTED');
CREATE TYPE validation_status AS ENUM ('PENDING', 'VALIDATED', 'FAILED_VALIDATION', 'CORRECTED');

-- Users Table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(50) UNIQUE,
    status user_status NOT NULL DEFAULT 'PENDING_VERIFICATION',
    role VARCHAR(50) NOT NULL DEFAULT 'POLICYHOLDER', -- 'POLICYHOLDER', 'REVIEWER', 'SENIOR_REVIEWER', 'PARTNER', 'ADMIN'
    preferred_language VARCHAR(10) NOT NULL DEFAULT 'en',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- User Sessions Table
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token VARCHAR(500) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Cases Table
CREATE TABLE cases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_number VARCHAR(100) UNIQUE NOT NULL,
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    insurance_type VARCHAR(100) NOT NULL, -- e.g., 'HEALTH'
    claim_category VARCHAR(100), -- e.g., 'ROOM_RENT', 'CO-PAYMENT', 'PRE-EXISTING'
    claim_status VARCHAR(100), -- e.g., 'REJECTED', 'PARTIALLY_SETTLED'
    insurer_name VARCHAR(255),
    policy_number_encrypted TEXT,
    claim_number_encrypted TEXT,
    amount_claimed NUMERIC(15, 2) DEFAULT 0.00,
    amount_paid NUMERIC(15, 2) DEFAULT 0.00,
    amount_disputed NUMERIC(15, 2) DEFAULT 0.00,
    risk_level risk_level NOT NULL DEFAULT 'LOW',
    workflow_state workflow_state NOT NULL DEFAULT 'DRAFT',
    preferred_language VARCHAR(10) NOT NULL DEFAULT 'en',
    assigned_reviewer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMP WITH TIME ZONE
);

-- Case Status History (Workflow Timeline)
CREATE TABLE case_status_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    from_state workflow_state NOT NULL,
    to_state workflow_state NOT NULL,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- User Consents
CREATE TABLE consents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    consent_version VARCHAR(50) NOT NULL,
    document_processing_consent BOOLEAN NOT NULL DEFAULT FALSE,
    reviewer_access_consent BOOLEAN NOT NULL DEFAULT FALSE,
    data_retention_consent BOOLEAN NOT NULL DEFAULT FALSE,
    authority_confirmation BOOLEAN NOT NULL DEFAULT FALSE,
    research_consent BOOLEAN NOT NULL DEFAULT FALSE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    withdrawn_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Documents Table
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    document_type VARCHAR(100) NOT NULL, -- e.g. 'REJECTION_LETTER', 'POLICY_SCHEDULE', 'DISCHARGE_SUMMARY'
    original_filename VARCHAR(255) NOT NULL,
    storage_key VARCHAR(500) NOT NULL,
    file_hash VARCHAR(64) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL,
    page_count INTEGER NOT NULL DEFAULT 0,
    malware_scan_status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- 'CLEAN', 'INFECTED', 'PENDING'
    ocr_status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'RUNNING', 'COMPLETED', 'FAILED'
    classification_status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'COMPLETED', 'FAILED'
    retention_until TIMESTAMP WITH TIME ZONE,
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Document Pages (if split or tracked per page)
CREATE TABLE document_pages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    page_number INTEGER NOT NULL,
    storage_key VARCHAR(500) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Document Extractions
CREATE TABLE document_extractions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    field_name VARCHAR(100) NOT NULL, -- e.g. 'claim_number', 'amount_claimed', 'hospital_name'
    field_value TEXT,
    normalized_value TEXT,
    page_number INTEGER,
    source_text TEXT,
    bounding_box JSONB, -- Coordinates [x1, y1, x2, y2]
    confidence NUMERIC(5, 4) DEFAULT 0.0000,
    extractor_version VARCHAR(50),
    review_status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'APPROVED', 'CORRECTED'
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Knowledge Base / Regulatory Corpus
CREATE TABLE knowledge_sources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_type VARCHAR(50) NOT NULL, -- 'POLICY_WORDING', 'REGULATION' (IRDAI)
    insurer_name VARCHAR(255),
    product_name VARCHAR(255),
    version VARCHAR(50),
    effective_date DATE,
    superseded_by_id UUID,
    title VARCHAR(500) NOT NULL,
    file_storage_key VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Knowledge Chunks (For RAG vector search)
CREATE TABLE knowledge_chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    knowledge_source_id UUID NOT NULL REFERENCES knowledge_sources(id) ON DELETE CASCADE,
    clause_number VARCHAR(100),
    heading TEXT,
    content TEXT NOT NULL,
    page_number INTEGER,
    embedding vector(384), -- Using sentence-transformers miniLM-L6-v2 (384 dimensions)
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Case Issues / Findings (e.g. Room Rent Deduction, Pre-existing illness)
CREATE TABLE case_issues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    issue_category VARCHAR(100) NOT NULL, -- 'ROOM_RENT_DEDUCTION', 'CO-PAYMENT', 'PRE-EXISTING'
    summary TEXT NOT NULL,
    details JSONB,
    confidence NUMERIC(5, 4) DEFAULT 0.0000,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Citations
CREATE TABLE citations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    source_type VARCHAR(50) NOT NULL, -- 'POLICY', 'REGULATION'
    document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    knowledge_source_id UUID REFERENCES knowledge_sources(id) ON DELETE SET NULL,
    page_number INTEGER,
    section_name VARCHAR(255),
    clause_number VARCHAR(100),
    quoted_text TEXT,
    quoted_text_hash VARCHAR(64),
    bounding_box JSONB,
    confidence NUMERIC(5, 4) DEFAULT 0.0000,
    validation_status validation_status NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Clarification Questions
CREATE TABLE clarification_questions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    question_type VARCHAR(50) NOT NULL, -- 'YES_NO', 'DATE', 'AMOUNT', 'DOCUMENT_UPLOAD', 'TEXT', 'MULTIPLE_CHOICE'
    question_text TEXT NOT NULL,
    options JSONB, -- For multiple choice
    context_explanation TEXT,
    source_document_ref TEXT,
    is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Clarification Answers
CREATE TABLE clarification_answers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    question_id UUID NOT NULL REFERENCES clarification_questions(id) ON DELETE CASCADE,
    answer_text TEXT NOT NULL,
    uploaded_evidence_document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    answered_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Evidence Items checklist
CREATE TABLE evidence_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    document_name VARCHAR(255) NOT NULL,
    why_required TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'MEDIUM', -- 'HIGH', 'MEDIUM', 'LOW'
    is_mandatory BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(50) NOT NULL DEFAULT 'MISSING', -- 'AVAILABLE', 'MISSING', 'CONTRADICTORY', 'RECOMMENDED', 'REQUESTED'
    uploaded_document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Drafts
CREATE TABLE drafts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT', -- 'DRAFT', 'APPROVED', 'REJECTED'
    current_version INTEGER NOT NULL DEFAULT 1,
    safety_status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'PASS', 'WARNING', 'BLOCK'
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Draft Versions
CREATE TABLE draft_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    draft_id UUID NOT NULL REFERENCES drafts(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    subject TEXT NOT NULL,
    content TEXT NOT NULL, -- Rich text content (HTML or JSON representing TipTap schema)
    meta_details JSONB,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Reviews
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    decision VARCHAR(50) NOT NULL, -- 'APPROVED', 'REJECTED', 'NEEDS_INFO', 'ESCALATED'
    risk_override VARCHAR(50),
    comments TEXT,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    sla_due_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Review Comments
CREATE TABLE review_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_text TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Submissions Tracking
CREATE TABLE submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    submission_channel VARCHAR(50) NOT NULL, -- 'EMAIL', 'PORTAL', 'POST', 'NGO'
    submission_date DATE NOT NULL,
    reference_number VARCHAR(100),
    acknowledgment_storage_key VARCHAR(500),
    response_date DATE,
    next_escalation_stage VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'SUBMITTED', -- 'SUBMITTED', 'ACKNOWLEDGED', 'RESPONSED', 'CLOSED'
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Model Runs Logging (Audit / Cost estimation)
CREATE TABLE model_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID REFERENCES cases(id) ON DELETE SET NULL,
    task_type VARCHAR(100) NOT NULL, -- 'CLASSIFICATION', 'EXTRACTION', 'REASONING', 'DRAFTING'
    provider VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    prompt_version VARCHAR(50),
    input_hash VARCHAR(64),
    output_hash VARCHAR(64),
    token_usage JSONB, -- {prompt_tokens: 12, completion_tokens: 24, total_tokens: 36}
    cost NUMERIC(10, 6) DEFAULT 0.000000,
    latency_ms INTEGER DEFAULT 0,
    status VARCHAR(50) NOT NULL, -- 'SUCCESS', 'FAILED'
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Audit Events (Immutable Append-only log)
CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_role VARCHAR(50),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    before_hash VARCHAR(64),
    after_hash VARCHAR(64),
    ip_address VARCHAR(45),
    user_agent TEXT,
    correlation_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
