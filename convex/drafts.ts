import { query, mutation } from "./_generated/server";
import { v } from "convex/values";

export const listByCase = query({
  args: { caseId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db.query("drafts")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
  },
});

export const getByLegacyId = query({
  args: { legacyId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db.query("drafts")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.legacyId))
      .first();
  },
});

export const create = mutation({
  args: {
    caseId: v.string(),
    language: v.string(),
    status: v.string(),
    currentVersion: v.number(),
    safetyStatus: v.string(),
    createdBy: v.optional(v.string()),
    legacyId: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const now = new Date().toISOString();
    return await ctx.db.insert("drafts", { ...args, createdAt: now, updatedAt: now });
  },
});

export const update = mutation({
  args: {
    legacyId: v.string(),
    currentVersion: v.optional(v.number()),
    status: v.optional(v.string()),
    safetyStatus: v.optional(v.string()),
    approvedBy: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const draft = await ctx.db.query("drafts")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.legacyId))
      .first();
    if (!draft) throw new Error(`Draft not found: ${args.legacyId}`);
    const updates: Record<string, unknown> = { updatedAt: new Date().toISOString() };
    if (args.currentVersion !== undefined) updates.currentVersion = args.currentVersion;
    if (args.status !== undefined) updates.status = args.status;
    if (args.safetyStatus !== undefined) updates.safetyStatus = args.safetyStatus;
    if (args.approvedBy !== undefined) updates.approvedBy = args.approvedBy;
    await ctx.db.patch(draft._id, updates);
    return draft._id;
  },
});

export const createVersion = mutation({
  args: {
    draftId: v.string(),
    versionNumber: v.number(),
    subject: v.string(),
    content: v.string(),
    createdBy: v.optional(v.string()),
    legacyId: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    return await ctx.db.insert("draftVersions", { ...args, createdAt: new Date().toISOString() });
  },
});

export const listVersions = query({
  args: { draftId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db.query("draftVersions")
      .withIndex("by_draft_id", (q) => q.eq("draftId", args.draftId))
      .collect();
  },
});

export const getVersion = query({
  args: { draftId: v.string(), versionNumber: v.number() },
  handler: async (ctx, args) => {
    const versions = await ctx.db.query("draftVersions")
      .withIndex("by_draft_id", (q) => q.eq("draftId", args.draftId))
      .collect();
    return versions.find((ver) => ver.versionNumber === args.versionNumber) || null;
  },
});
