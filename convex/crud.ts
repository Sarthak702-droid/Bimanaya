import { query, mutation } from "./_generated/server";
import { v } from "convex/values";

export const getRecord = query({
  args: {
    table: v.string(),
    field: v.string(),
    value: v.string(),
  },
  handler: async (ctx, args) => {
    const table = args.table as any;
    try {
      return await ctx.db
        .query(table)
        .withIndex("by_" + args.field, (q: any) => q.eq(args.field, args.value))
        .first();
    } catch (e) {
      return await ctx.db
        .query(table)
        .filter((q) => q.eq(q.field(args.field), args.value))
        .first();
    }
  },
});

export const listRecords = query({
  args: {
    table: v.string(),
    field: v.optional(v.string()),
    value: v.optional(v.any()),
  },
  handler: async (ctx, args) => {
    const table = args.table as any;
    if (!args.field || args.value === undefined) {
      return await ctx.db.query(table).collect();
    }
    try {
      return await ctx.db
        .query(table)
        .withIndex("by_" + args.field, (q: any) => q.eq(args.field, args.value))
        .collect();
    } catch (e) {
      return await ctx.db
        .query(table)
        .filter((q) => q.eq(q.field(args.field!), args.value))
        .collect();
    }
  },
});

export const insertRecord = mutation({
  args: {
    table: v.string(),
    data: v.any(),
  },
  handler: async (ctx, args) => {
    return await ctx.db.insert(args.table as any, args.data);
  },
});

export const updateRecord = mutation({
  args: {
    table: v.string(),
    idField: v.string(),
    idValue: v.string(),
    updates: v.any(),
  },
  handler: async (ctx, args) => {
    const table = args.table as any;
    const doc = await ctx.db
      .query(table)
      .filter((q) => q.eq(q.field(args.idField), args.idValue))
      .first();
    if (!doc) {
      throw new Error(`Record not found in ${args.table} for ${args.idField} = ${args.idValue}`);
    }
    await ctx.db.patch(doc._id, args.updates);
    return doc._id;
  },
});

export const deleteRecord = mutation({
  args: {
    table: v.string(),
    idField: v.string(),
    idValue: v.string(),
  },
  handler: async (ctx, args) => {
    const table = args.table as any;
    const doc = await ctx.db
      .query(table)
      .filter((q) => q.eq(q.field(args.idField), args.idValue))
      .first();
    if (doc) {
      await ctx.db.delete(doc._id);
      return true;
    }
    return false;
  },
});
