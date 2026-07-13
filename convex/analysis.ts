import { query, mutation } from "./_generated/server";
import { v } from "convex/values";

export const getAnalysis = query({
  args: { caseId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db.query("caseIssues")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
  },
});

export const getCitations = query({
  args: { caseId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db.query("citations")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
  },
});

export const getEvidence = query({
  args: { caseId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db.query("evidenceItems")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
  },
});

export const saveAnalysisFindings = mutation({
  args: {
    caseId: v.string(),
    issues: v.array(
      v.object({
        issueCategory: v.string(),
        summary: v.string(),
        details: v.optional(v.string()),
        confidence: v.number(),
        legacyId: v.optional(v.string()),
      })
    ),
    citations: v.array(
      v.object({
        sourceType: v.string(),
        documentId: v.optional(v.string()),
        knowledgeSourceId: v.optional(v.string()),
        pageNumber: v.optional(v.number()),
        sectionName: v.optional(v.string()),
        clauseNumber: v.optional(v.string()),
        quotedText: v.optional(v.string()),
        boundingBox: v.optional(v.string()),
        confidence: v.number(),
        validationStatus: v.union(v.literal("PENDING"), v.literal("VALIDATED"), v.literal("FAILED_VALIDATION"), v.literal("CORRECTED")),
        legacyId: v.optional(v.string()),
      })
    ),
    evidenceItems: v.array(
      v.object({
        documentName: v.string(),
        whyRequired: v.string(),
        priority: v.string(),
        isMandatory: v.boolean(),
        status: v.string(),
        uploadedDocumentId: v.optional(v.string()),
        legacyId: v.optional(v.string()),
      })
    ),
    changedBy: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    // 1. Delete old findings
    const oldIssues = await ctx.db.query("caseIssues")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
    for (const issue of oldIssues) {
      await ctx.db.delete(issue._id);
    }

    const oldCitations = await ctx.db.query("citations")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
    for (const citation of oldCitations) {
      await ctx.db.delete(citation._id);
    }

    const oldEvidence = await ctx.db.query("evidenceItems")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
    for (const item of oldEvidence) {
      await ctx.db.delete(item._id);
    }

    // 2. Insert new findings
    const now = new Date().toISOString();
    for (const issue of args.issues) {
      await ctx.db.insert("caseIssues", {
        caseId: args.caseId,
        ...issue,
        createdAt: now,
      });
    }

    for (const citation of args.citations) {
      await ctx.db.insert("citations", {
        caseId: args.caseId,
        ...citation,
        createdAt: now,
      });
    }

    for (const item of args.evidenceItems) {
      await ctx.db.insert("evidenceItems", {
        caseId: args.caseId,
        ...item,
        createdAt: now,
      });
    }

    // 3. Update case status to ANALYSIS_READY or REVIEW_REQUIRED
    const caseItem = await ctx.db.query("cases")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.caseId))
      .first();

    if (caseItem) {
      await ctx.db.patch(caseItem._id, {
        workflowState: "ANALYSIS_READY",
        updatedAt: now,
      });

      await ctx.db.insert("caseStatusHistory", {
        caseId: args.caseId,
        fromState: "PROCESSING",
        toState: "ANALYSIS_READY",
        changedBy: args.changedBy,
        reason: "AI analysis completed",
        createdAt: now,
      });
    }

    return true;
  },
});
