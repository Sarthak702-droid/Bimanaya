import { query, mutation } from "./_generated/server";
import { v } from "convex/values";
import { validateCaseAccess } from "./authHelpers";

// ── List documents by case ──────────────────────────────────────────────
export const listByCase = query({
  args: { caseId: v.string() },
  handler: async (ctx, args) => {
    await validateCaseAccess(ctx, args.caseId);
    const docs = await ctx.db
      .query("documents")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
    return docs.filter((d) => !d.deletedAt);
  },
});

// ── Get document by legacy ID ───────────────────────────────────────────
export const getByLegacyId = query({
  args: { legacyId: v.string() },
  handler: async (ctx, args) => {
    const doc = await ctx.db
      .query("documents")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.legacyId))
      .first();
    
    if (!doc) return null;
    await validateCaseAccess(ctx, doc.caseId);
    return doc;
  },
});

// ── Create document metadata ────────────────────────────────────────────
export const createMetadata = mutation({
  args: {
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
    uploadedBy: v.optional(v.string()),
    legacyId: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    await validateCaseAccess(ctx, args.caseId, ["POLICYHOLDER", "REVIEWER", "SENIOR_REVIEWER", "ADMIN"], args.uploadedBy);
    return await ctx.db.insert("documents", {
      ...args,
      createdAt: new Date().toISOString(),
    });
  },
});

// ── Update document status ──────────────────────────────────────────────
export const updateStatus = mutation({
  args: {
    legacyId: v.string(),
    fileHash: v.optional(v.string()),
    pageCount: v.optional(v.number()),
    malwareScanStatus: v.optional(v.string()),
    ocrStatus: v.optional(v.string()),
    classificationStatus: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const doc = await ctx.db
      .query("documents")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.legacyId))
      .first();

    if (!doc) {
      throw new Error(`Document not found for legacyId: ${args.legacyId}`);
    }

    await validateCaseAccess(ctx, doc.caseId);

    const updates: Record<string, unknown> = {};
    if (args.fileHash !== undefined) updates.fileHash = args.fileHash;
    if (args.pageCount !== undefined) updates.pageCount = args.pageCount;
    if (args.malwareScanStatus !== undefined) updates.malwareScanStatus = args.malwareScanStatus;
    if (args.ocrStatus !== undefined) updates.ocrStatus = args.ocrStatus;
    if (args.classificationStatus !== undefined) updates.classificationStatus = args.classificationStatus;

    await ctx.db.patch(doc._id, updates);
    return doc._id;
  },
});

// ── Soft delete document ────────────────────────────────────────────────
export const softDelete = mutation({
  args: { legacyId: v.string() },
  handler: async (ctx, args) => {
    const doc = await ctx.db
      .query("documents")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.legacyId))
      .first();

    if (!doc) {
      return false;
    }

    await validateCaseAccess(ctx, doc.caseId);

    await ctx.db.patch(doc._id, {
      deletedAt: new Date().toISOString(),
    });
    return true;
  },
});

// ── Save Document Extraction ───────────────────────────────────────────
export const saveExtraction = mutation({
  args: {
    documentId: v.string(),
    fieldName: v.string(),
    fieldValue: v.optional(v.string()),
    normalizedValue: v.optional(v.string()),
    pageNumber: v.optional(v.number()),
    sourceText: v.optional(v.string()),
    confidence: v.number(),
    reviewStatus: v.string(),
    legacyId: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    // Find the document to check access on its parent case
    let doc = await ctx.db
      .query("documents")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.documentId))
      .first();
    
    if (!doc) {
      try {
        doc = await ctx.db.get(args.documentId as any);
      } catch (e) {}
    }

    if (doc) {
      await validateCaseAccess(ctx, doc.caseId);
    }

    return await ctx.db.insert("documentExtractions", {
      ...args,
      createdAt: new Date().toISOString(),
    });
  },
});

// ── Save Document Page ──────────────────────────────────────────────────
export const savePage = mutation({
  args: {
    documentId: v.string(),
    pageNumber: v.number(),
    storageKey: v.string(),
    legacyId: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    // Find the document to check access on its parent case
    let doc = await ctx.db
      .query("documents")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.documentId))
      .first();
    
    if (!doc) {
      try {
        doc = await ctx.db.get(args.documentId as any);
      } catch (e) {}
    }

    if (doc) {
      await validateCaseAccess(ctx, doc.caseId);
    }

    return await ctx.db.insert("documentPages", {
      ...args,
      createdAt: new Date().toISOString(),
    });
  },
});

// ── Update Document Type & Status ───────────────────────────────────────
export const updateTypeAndStatus = mutation({
  args: {
    legacyId: v.string(),
    documentType: v.string(),
    ocrStatus: v.string(),
    classificationStatus: v.string(),
  },
  handler: async (ctx, args) => {
    const doc = await ctx.db
      .query("documents")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.legacyId))
      .first();

    if (!doc) {
      throw new Error(`Document not found for legacyId: ${args.legacyId}`);
    }

    await validateCaseAccess(ctx, doc.caseId);

    await ctx.db.patch(doc._id, {
      documentType: args.documentType,
      ocrStatus: args.ocrStatus,
      classificationStatus: args.classificationStatus,
    });
    return doc._id;
  },
});
