import { query, mutation } from "./_generated/server";
import { v } from "convex/values";
import { getCurrentUserOrThrow, requireRole, validateCaseAccess } from "./authHelpers";

// ── List cases for current user ─────────────────────────────────────────
export const listForUser = query({
  args: { ownerUserId: v.string() },
  handler: async (ctx, args) => {
    const user = await getCurrentUserOrThrow(ctx, args.ownerUserId);
    
    // Find cases by user._id or user.legacyId or user.clerkUserId using indexes
    const casesById = await ctx.db
      .query("cases")
      .withIndex("by_owner_user_id", (q) => q.eq("ownerUserId", user._id))
      .collect();
    
    const casesByLegacyId = user.legacyId ? await ctx.db
      .query("cases")
      .withIndex("by_owner_user_id", (q) => q.eq("ownerUserId", user.legacyId!))
      .collect() : [];

    const casesByClerkId = await ctx.db
      .query("cases")
      .withIndex("by_owner_user_id", (q) => q.eq("ownerUserId", user.clerkUserId))
      .collect();

    // Deduplicate and return
    const all = [...casesById, ...casesByLegacyId, ...casesByClerkId];
    const seen = new Set();
    return all.filter(c => {
      if (seen.has(c._id)) return false;
      seen.add(c._id);
      return true;
    });
  },
});

// ── List all cases (admin/reviewer) ─────────────────────────────────────
export const listAll = query({
  args: {},
  handler: async (ctx) => {
    await requireRole(ctx, ["REVIEWER", "SENIOR_REVIEWER", "ADMIN"]);
    return await ctx.db.query("cases").collect();
  },
});

// ── List cases by workflow state ────────────────────────────────────────
export const listByWorkflowState = query({
  args: { states: v.array(v.string()) },
  handler: async (ctx, args) => {
    await requireRole(ctx, ["REVIEWER", "SENIOR_REVIEWER", "ADMIN"]);
    
    // Fetch cases for each state in parallel using the index instead of a full table scan
    const promises = args.states.map(state => 
      ctx.db
        .query("cases")
        .withIndex("by_workflow_state", (q) => q.eq("workflowState", state as any))
        .collect()
    );
    const results = await Promise.all(promises);
    return results.flat();
  },
});

// ── Get case by ID (legacy string ID) ───────────────────────────────────
export const getByLegacyId = query({
  args: { legacyId: v.string() },
  handler: async (ctx, args) => {
    const { caseObj } = await validateCaseAccess(ctx, args.legacyId);
    return caseObj;
  },
});

// ── Get case by case number ─────────────────────────────────────────────
export const getByCaseNumber = query({
  args: { caseNumber: v.string() },
  handler: async (ctx, args) => {
    const caseObj = await ctx.db
      .query("cases")
      .withIndex("by_case_number", (q) => q.eq("caseNumber", args.caseNumber))
      .first();

    if (!caseObj) return null;

    // Validate access to the found case
    await validateCaseAccess(ctx, caseObj.legacyId || caseObj._id);
    return caseObj;
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
    // Validate authenticated user
    const user = await getCurrentUserOrThrow(ctx, args.ownerUserId);

    const now = new Date().toISOString();
    return await ctx.db.insert("cases", {
      ...args,
      ownerUserId: user._id, // Store direct Convex document ID of the owner
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
    // Validate case access
    const { caseObj, user } = await validateCaseAccess(ctx, args.legacyId);

    // Enforce role constraints for update fields
    if (user.role === "POLICYHOLDER" && process.env.ENV !== "development") {
      if (args.riskLevel !== undefined || args.assignedReviewerId !== undefined || args.closedAt !== undefined) {
        throw new Error("FORBIDDEN: Policyholder cannot update reviewer-only fields");
      }
      if (args.workflowState !== undefined && 
          args.workflowState !== "DRAFT" && 
          args.workflowState !== "CONSENT_PENDING" && 
          args.workflowState !== "DOCUMENTS_PENDING") {
        throw new Error("FORBIDDEN: Policyholder cannot perform this state transition");
      }
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

    await ctx.db.patch(caseObj._id, updates);
    return caseObj._id;
  },
});

// ── Delete case (soft delete via workflow state) ────────────────────────
export const softDelete = mutation({
  args: { legacyId: v.string() },
  handler: async (ctx, args) => {
    const { caseObj } = await validateCaseAccess(ctx, args.legacyId);

    const now = new Date().toISOString();
    await ctx.db.patch(caseObj._id, {
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
    // Validate case access
    await validateCaseAccess(ctx, args.caseId);

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
    // Validate case access
    await validateCaseAccess(ctx, args.caseId);

    return await ctx.db.insert("caseStatusHistory", {
      ...args,
      createdAt: new Date().toISOString(),
    });
  },
});
