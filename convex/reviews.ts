import { query, mutation } from "./_generated/server";
import { v } from "convex/values";

export const listByCase = query({
  args: { caseId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db.query("reviews")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
  },
});

export const listByReviewer = query({
  args: { reviewerId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db.query("reviews")
      .withIndex("by_reviewer_id", (q) => q.eq("reviewerId", args.reviewerId))
      .collect();
  },
});

export const getBreachedReviews = query({
  args: { now: v.string() },
  handler: async (ctx, args) => {
    return (await ctx.db.query("reviews")
      .withIndex("by_decision", (q) => q.eq("decision", "CLAIMED"))
      .collect()
    ).filter((r) => !r.completedAt && r.slaDueAt < args.now);
  },
});

export const create = mutation({
  args: {
    caseId: v.string(),
    reviewerId: v.string(),
    decision: v.union(v.literal("CLAIMED"), v.literal("APPROVED"), v.literal("REJECTED"), v.literal("NEEDS_INFO"), v.literal("ESCALATED")),
    comments: v.optional(v.string()),
    slaDueAt: v.string(),
    legacyId: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    return await ctx.db.insert("reviews", {
      ...args,
      startedAt: new Date().toISOString(),
    });
  },
});

export const updateDecision = mutation({
  args: {
    caseId: v.string(),
    reviewerId: v.string(),
    decision: v.union(v.literal("CLAIMED"), v.literal("APPROVED"), v.literal("REJECTED"), v.literal("NEEDS_INFO"), v.literal("ESCALATED")),
    comments: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const reviews = await ctx.db.query("reviews")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
    const active = reviews.find((r) => r.reviewerId === args.reviewerId && !r.completedAt);
    if (!active) throw new Error("No active review found");
    await ctx.db.patch(active._id, {
      decision: args.decision,
      comments: args.comments,
      completedAt: new Date().toISOString(),
    });
    return active._id;
  },
});

export const addComment = mutation({
  args: { caseId: v.string(), reviewerId: v.string(), commentText: v.string(), legacyId: v.optional(v.string()) },
  handler: async (ctx, args) => {
    return await ctx.db.insert("reviewComments", { ...args, createdAt: new Date().toISOString() });
  },
});

export const listComments = query({
  args: { caseId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db.query("reviewComments")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
  },
});
