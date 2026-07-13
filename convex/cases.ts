import { query, mutation } from "./_generated/server";
import { v } from "convex/values";

// ── List cases for current user ─────────────────────────────────────────
export const listForUser = query({
  args: { ownerUserId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db
      .query("cases")
      .withIndex("by_owner_user_id", (q) => q.eq("ownerUserId", args.ownerUserId))
      .collect();
  },
});

// ── List all cases (admin/reviewer) ─────────────────────────────────────
export const listAll = query({
  args: {},
  handler: async (ctx) => {
    return await ctx.db.query("cases").collect();
  },
});

// ── List cases by workflow state ────────────────────────────────────────
export const listByWorkflowState = query({
  args: { states: v.array(v.string()) },
  handler: async (ctx, args) => {
    const allCases = await ctx.db.query("cases").collect();
    return allCases.filter((c) => args.states.includes(c.workflowState));
  },
});

// ── Get case by ID (legacy string ID) ───────────────────────────────────
export const getByLegacyId = query({
  args: { legacyId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db
      .query("cases")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.legacyId))
      .first();
  },
});

// ── Get case by case number ─────────────────────────────────────────────
export const getByCaseNumber = query({
  args: { caseNumber: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db
      .query("cases")
      .withIndex("by_case_number", (q) => q.eq("caseNumber", args.caseNumber))
      .first();
  },
});

// ── Create case ─────────────────────────────────────────────────────────
export const create = mutation({
  args: {
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
    preferredLanguage: v.string(),
    legacyId: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const now = new Date().toISOString();
    return await ctx.db.insert("cases", {
      ...args,
      riskLevel: "LOW",
      workflowState: "DRAFT",
      createdAt: now,
      updatedAt: now,
    });
  },
});

// ── Update case ─────────────────────────────────────────────────────────
export const update = mutation({
  args: {
    legacyId: v.string(),
    claimCategory: v.optional(v.string()),
    claimStatus: v.optional(v.string()),
    insurerName: v.optional(v.string()),
    amountClaimed: v.optional(v.number()),
    amountPaid: v.optional(v.number()),
    amountDisputed: v.optional(v.number()),
    workflowState: v.optional(v.string()),
    riskLevel: v.optional(v.string()),
    assignedReviewerId: v.optional(v.string()),
    closedAt: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const existing = await ctx.db
      .query("cases")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.legacyId))
      .first();

    if (!existing) {
      throw new Error(`Case not found for legacyId: ${args.legacyId}`);
    }

    const updates: Record<string, unknown> = { updatedAt: new Date().toISOString() };
    if (args.claimCategory !== undefined) updates.claimCategory = args.claimCategory;
    if (args.claimStatus !== undefined) updates.claimStatus = args.claimStatus;
    if (args.insurerName !== undefined) updates.insurerName = args.insurerName;
    if (args.amountClaimed !== undefined) updates.amountClaimed = args.amountClaimed;
    if (args.amountPaid !== undefined) updates.amountPaid = args.amountPaid;
    if (args.amountDisputed !== undefined) updates.amountDisputed = args.amountDisputed;
    if (args.workflowState !== undefined) updates.workflowState = args.workflowState;
    if (args.riskLevel !== undefined) updates.riskLevel = args.riskLevel;
    if (args.assignedReviewerId !== undefined) updates.assignedReviewerId = args.assignedReviewerId || undefined;
    if (args.closedAt !== undefined) updates.closedAt = args.closedAt;

    await ctx.db.patch(existing._id, updates);
    return existing._id;
  },
});

// ── Delete case (soft delete via workflow state) ────────────────────────
export const softDelete = mutation({
  args: { legacyId: v.string() },
  handler: async (ctx, args) => {
    const existing = await ctx.db
      .query("cases")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.legacyId))
      .first();

    if (!existing) {
      throw new Error(`Case not found for legacyId: ${args.legacyId}`);
    }

    const now = new Date().toISOString();
    await ctx.db.patch(existing._id, {
      workflowState: "DELETED",
      closedAt: now,
      updatedAt: now,
    });
    return true;
  },
});

// ── Get timeline for case ───────────────────────────────────────────────
export const getTimeline = query({
  args: { caseId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db
      .query("caseStatusHistory")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
  },
});

// ── Log timeline event ──────────────────────────────────────────────────
export const logTimeline = mutation({
  args: {
    caseId: v.string(),
    fromState: v.string(),
    toState: v.string(),
    changedBy: v.optional(v.string()),
    reason: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    return await ctx.db.insert("caseStatusHistory", {
      ...args,
      createdAt: new Date().toISOString(),
    });
  },
});
