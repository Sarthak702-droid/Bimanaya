import { query, mutation } from "./_generated/server";
import { v } from "convex/values";

export const list = query({
  args: { limit: v.optional(v.number()) },
  handler: async (ctx, args) => {
    const limit = args.limit || 100;
    return await ctx.db.query("auditEvents")
      .order("desc")
      .take(limit);
  },
});

export const log = mutation({
  args: {
    actorId: v.optional(v.string()),
    actorRole: v.optional(v.string()),
    actorType: v.optional(v.string()),
    action: v.string(),
    resourceType: v.string(),
    resourceId: v.optional(v.string()),
    beforeHash: v.optional(v.string()),
    afterHash: v.optional(v.string()),
    ipAddress: v.optional(v.string()),
    userAgent: v.optional(v.string()),
    correlationId: v.string(),
    legacyId: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    return await ctx.db.insert("auditEvents", {
      ...args,
      createdAt: new Date().toISOString(),
    });
  },
});
