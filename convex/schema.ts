import { defineSchema, defineTable } from "convex/server";
import { v } from "convex/values";

// ── Enum-like literal unions ────────────────────────────────────────────
const userRole = v.union(
  v.literal("POLICYHOLDER"),
  v.literal("FAMILY_MEMBER"),
  v.literal("REVIEWER"),
  v.literal("SENIOR_REVIEWER"),
  v.literal("PARTNER"),
  v.literal("ADMIN"),
  v.literal("OPERATIONS")
);

const userStatus = v.union(
  v.literal("ACTIVE"),
  v.literal("SUSPENDED"),
  v.literal("PENDING_VERIFICATION"),
  v.literal("DELETED")
);

const workflowState = v.union(
  v.literal("DRAFT"),
  v.literal("ELIGIBILITY_COMPLETED"),
  v.literal("CONSENT_PENDING"),
  v.literal("DOCUMENTS_PENDING"),
  v.literal("PROCESSING"),
  v.literal("NEEDS_CLARIFICATION"),
  v.literal("ANALYSIS_READY"),
  v.literal("REVIEW_REQUIRED"),
  v.literal("IN_REVIEW"),
  v.literal("MORE_INFORMATION_REQUIRED"),
  v.literal("APPROVED"),
  v.literal("READY_FOR_EXPORT"),
  v.literal("SUBMITTED"),
  v.literal("TRACKING"),
  v.literal("RESOLVED"),
  v.literal("CLOSED"),
  v.literal("DELETION_PENDING"),
  v.literal("DELETED")
);

const riskLevel = v.union(
  v.literal("LOW"),
  v.literal("MEDIUM"),
  v.literal("HIGH"),
  v.literal("CRITICAL")
);

const reviewDecision = v.union(
  v.literal("CLAIMED"),
  v.literal("APPROVED"),
  v.literal("REJECTED"),
  v.literal("NEEDS_INFO"),
  v.literal("ESCALATED")
);

const validationStatus = v.union(
  v.literal("PENDING"),
  v.literal("VALIDATED"),
  v.literal("FAILED_VALIDATION"),
  v.literal("CORRECTED")
);

// ── Schema ──────────────────────────────────────────────────────────────
export default defineSchema({
  // ── Users ─────────────────────────────────────────────────────────────
  // Clerk is the source of truth for identity.
  // Convex is the source of truth for application profile and role data.
  users: defineTable({
    // Clerk identity fields
    clerkUserId: v.string(),
    clerkSubject: v.string(),
    email: v.string(),
    emailVerified: v.boolean(),
    firstName: v.optional(v.string()),
    lastName: v.optional(v.string()),
    displayName: v.optional(v.string()),
    imageUrl: v.optional(v.string()),
    phone: v.optional(v.string()),
    // Application fields
    preferredLanguage: v.string(),
    role: userRole,
    status: userStatus,
    onboardingCompleted: v.boolean(),
    lastLoginAt: v.string(),
    // Legacy migration fields
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
    updatedAt: v.string(),
    deletedAt: v.optional(v.string()),
  })
    .index("by_clerk_user_id", ["clerkUserId"])
    .index("by_clerk_subject", ["clerkSubject"])
    .index("by_email", ["email"])
    .index("by_role", ["role"])
    .index("by_status", ["status"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Cases ─────────────────────────────────────────────────────────────
  cases: defineTable({
    caseNumber: v.string(),
    ownerUserId: v.string(),
    insuranceType: v.string(),
    claimCategory: v.optional(v.string()),
    claimStatus: v.optional(v.string()),
    insurerName: v.optional(v.string()),
    policyNumberEncrypted: v.optional(v.string()),
    claimNumberEncrypted: v.optional(v.string()),
    amountClaimed: v.number(),
    amountPaid: v.number(),
    amountDisputed: v.number(),
    riskLevel: riskLevel,
    workflowState: workflowState,
    preferredLanguage: v.string(),
    assignedReviewerId: v.optional(v.string()),
    // Legacy migration fields
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
    updatedAt: v.string(),
    closedAt: v.optional(v.string()),
  })
    .index("by_case_number", ["caseNumber"])
    .index("by_owner_user_id", ["ownerUserId"])
    .index("by_assigned_reviewer_id", ["assignedReviewerId"])
    .index("by_workflow_state", ["workflowState"])
    .index("by_risk_level", ["riskLevel"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Case Status History ───────────────────────────────────────────────
  caseStatusHistory: defineTable({
    caseId: v.string(),
    fromState: v.string(),
    toState: v.string(),
    changedBy: v.optional(v.string()),
    reason: v.optional(v.string()),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_case_id", ["caseId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Consents ──────────────────────────────────────────────────────────
  consents: defineTable({
    caseId: v.string(),
    userId: v.string(),
    consentVersion: v.string(),
    documentProcessingConsent: v.boolean(),
    reviewerAccessConsent: v.boolean(),
    dataRetentionConsent: v.boolean(),
    authorityConfirmation: v.boolean(),
    researchConsent: v.boolean(),
    ipAddress: v.optional(v.string()),
    userAgent: v.optional(v.string()),
    withdrawnAt: v.optional(v.string()),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_case_id", ["caseId"])
    .index("by_user_id", ["userId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Documents ─────────────────────────────────────────────────────────
  documents: defineTable({
    caseId: v.string(),
    documentType: v.string(),
    originalFilename: v.string(),
    storageKey: v.string(),
    fileHash: v.string(),
    mimeType: v.string(),
    sizeBytes: v.number(),
    pageCount: v.number(),
    malwareScanStatus: v.string(),
    ocrStatus: v.string(),
    classificationStatus: v.string(),
    retentionUntil: v.optional(v.string()),
    uploadedBy: v.optional(v.string()),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
    deletedAt: v.optional(v.string()),
  })
    .index("by_case_id", ["caseId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Document Pages ────────────────────────────────────────────────────
  documentPages: defineTable({
    documentId: v.string(),
    pageNumber: v.number(),
    storageKey: v.string(),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_document_id", ["documentId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Document Extractions ──────────────────────────────────────────────
  documentExtractions: defineTable({
    documentId: v.string(),
    fieldName: v.string(),
    fieldValue: v.optional(v.string()),
    normalizedValue: v.optional(v.string()),
    pageNumber: v.optional(v.number()),
    sourceText: v.optional(v.string()),
    boundingBox: v.optional(v.string()),
    confidence: v.number(),
    extractorVersion: v.optional(v.string()),
    reviewStatus: v.string(),
    reviewedBy: v.optional(v.string()),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_document_id", ["documentId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Knowledge Sources ─────────────────────────────────────────────────
  knowledgeSources: defineTable({
    sourceType: v.string(),
    insurerName: v.optional(v.string()),
    productName: v.optional(v.string()),
    version: v.optional(v.string()),
    effectiveDate: v.optional(v.string()),
    supersededById: v.optional(v.string()),
    title: v.string(),
    fileStorageKey: v.optional(v.string()),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_source_type", ["sourceType"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Knowledge Chunks (RAG) ────────────────────────────────────────────
  knowledgeChunks: defineTable({
    knowledgeSourceId: v.string(),
    clauseNumber: v.optional(v.string()),
    heading: v.optional(v.string()),
    content: v.string(),
    pageNumber: v.optional(v.number()),
    embedding: v.optional(v.array(v.number())),
    metadata: v.optional(v.string()),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_knowledge_source_id", ["knowledgeSourceId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Case Issues ───────────────────────────────────────────────────────
  caseIssues: defineTable({
    caseId: v.string(),
    issueCategory: v.string(),
    summary: v.string(),
    details: v.optional(v.string()),
    confidence: v.number(),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_case_id", ["caseId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Citations ─────────────────────────────────────────────────────────
  citations: defineTable({
    caseId: v.string(),
    sourceType: v.string(),
    documentId: v.optional(v.string()),
    knowledgeSourceId: v.optional(v.string()),
    pageNumber: v.optional(v.number()),
    sectionName: v.optional(v.string()),
    clauseNumber: v.optional(v.string()),
    quotedText: v.optional(v.string()),
    quotedTextHash: v.optional(v.string()),
    boundingBox: v.optional(v.string()),
    confidence: v.number(),
    validationStatus: validationStatus,
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_case_id", ["caseId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Clarification Questions ───────────────────────────────────────────
  clarificationQuestions: defineTable({
    caseId: v.string(),
    questionType: v.string(),
    questionText: v.string(),
    options: v.optional(v.string()),
    contextExplanation: v.optional(v.string()),
    sourceDocumentRef: v.optional(v.string()),
    isResolved: v.boolean(),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_case_id", ["caseId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Clarification Answers ─────────────────────────────────────────────
  clarificationAnswers: defineTable({
    questionId: v.string(),
    answerText: v.string(),
    uploadedEvidenceDocumentId: v.optional(v.string()),
    answeredBy: v.string(),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_question_id", ["questionId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Evidence Items ────────────────────────────────────────────────────
  evidenceItems: defineTable({
    caseId: v.string(),
    documentName: v.string(),
    whyRequired: v.string(),
    priority: v.string(),
    isMandatory: v.boolean(),
    status: v.string(),
    uploadedDocumentId: v.optional(v.string()),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_case_id", ["caseId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Drafts ────────────────────────────────────────────────────────────
  drafts: defineTable({
    caseId: v.string(),
    language: v.string(),
    status: v.string(),
    currentVersion: v.number(),
    safetyStatus: v.string(),
    createdBy: v.optional(v.string()),
    approvedBy: v.optional(v.string()),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
    updatedAt: v.string(),
  })
    .index("by_case_id", ["caseId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Draft Versions ────────────────────────────────────────────────────
  draftVersions: defineTable({
    draftId: v.string(),
    versionNumber: v.number(),
    subject: v.string(),
    content: v.string(),
    metaDetails: v.optional(v.string()),
    createdBy: v.optional(v.string()),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_draft_id", ["draftId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Reviews ───────────────────────────────────────────────────────────
  reviews: defineTable({
    caseId: v.string(),
    reviewerId: v.string(),
    decision: reviewDecision,
    riskOverride: v.optional(v.string()),
    comments: v.optional(v.string()),
    startedAt: v.string(),
    completedAt: v.optional(v.string()),
    slaDueAt: v.string(),
    legacyId: v.optional(v.string()),
  })
    .index("by_case_id", ["caseId"])
    .index("by_reviewer_id", ["reviewerId"])
    .index("by_decision", ["decision"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Review Comments ───────────────────────────────────────────────────
  reviewComments: defineTable({
    caseId: v.string(),
    reviewerId: v.string(),
    commentText: v.string(),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_case_id", ["caseId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Submissions ───────────────────────────────────────────────────────
  submissions: defineTable({
    caseId: v.string(),
    submissionChannel: v.string(),
    submissionDate: v.string(),
    referenceNumber: v.optional(v.string()),
    acknowledgmentStorageKey: v.optional(v.string()),
    responseDate: v.optional(v.string()),
    nextEscalationStage: v.optional(v.string()),
    status: v.string(),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_case_id", ["caseId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Model Runs ────────────────────────────────────────────────────────
  modelRuns: defineTable({
    caseId: v.optional(v.string()),
    taskType: v.string(),
    provider: v.string(),
    model: v.string(),
    promptVersion: v.optional(v.string()),
    inputHash: v.optional(v.string()),
    outputHash: v.optional(v.string()),
    tokenUsage: v.optional(v.string()),
    cost: v.number(),
    latencyMs: v.number(),
    status: v.string(),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_case_id", ["caseId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Audit Events (Immutable) ──────────────────────────────────────────
  auditEvents: defineTable({
    actorId: v.optional(v.string()),
    actorRole: v.optional(v.string()),
    actorType: v.optional(v.string()), // "USER" | "SYSTEM"
    action: v.string(),
    resourceType: v.string(),
    resourceId: v.optional(v.string()),
    beforeHash: v.optional(v.string()),
    afterHash: v.optional(v.string()),
    ipAddress: v.optional(v.string()),
    userAgent: v.optional(v.string()),
    correlationId: v.string(),
    legacyId: v.optional(v.string()),
    createdAt: v.string(),
  })
    .index("by_actor_id", ["actorId"])
    .index("by_resource", ["resourceType", "resourceId"])
    .index("by_correlation_id", ["correlationId"])
    .index("by_legacy_id", ["legacyId"]),

  // ── Eligibility Assessments ───────────────────────────────────────────
  eligibilityAssessments: defineTable({
    userId: v.string(),
    insuranceType: v.string(),
    claimStatus: v.string(),
    disputedAmount: v.number(),
    availableDocuments: v.array(v.string()),
    userAuthority: v.boolean(),
    resultStatus: v.string(),
    missingDocuments: v.array(v.string()),
    manualReviewRequired: v.boolean(),
    reasonCodes: v.array(v.string()),
    createdAt: v.string(),
  })
    .index("by_user_id", ["userId"]),
});
