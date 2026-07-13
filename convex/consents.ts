import { query, mutation } from "./_generated/server";
import { v } from "convex/values";

// ── Record consent ──────────────────────────────────────────────────────
export const record = mutation({
  args: {
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
    legacyId: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    return await ctx.db.insert("consents", {
      ...args,
      createdAt: new Date().toISOString(),
    });
  },
});

// ── List consents by case ───────────────────────────────────────────────
export const listByCase = query({
  args: { caseId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db
      .query("consents")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
  },
});

// ── Withdraw consent ────────────────────────────────────────────────────
export const withdraw = mutation({
  args: { caseId: v.string() },
  handler: async (ctx, args) => {
    const consents = await ctx.db
      .query("consents")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();

    const now = new Date().toISOString();
    for (const c of consents) {
      if (!c.withdrawnAt) {
        await ctx.db.patch(c._id, { withdrawnAt: now });
      }
    }
    return true;
  },
});
