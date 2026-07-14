import { query, mutation } from "./_generated/server";
import { v } from "convex/values";
import { validateCaseAccess } from "./authHelpers";

export const getQuestions = query({
  args: { caseId: v.string() },
  handler: async (ctx, args) => {
    await validateCaseAccess(ctx, args.caseId);
    return await ctx.db.query("clarificationQuestions")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();
  },
});

export const submitAnswer = mutation({
  args: {
    caseId: v.string(),
    questionId: v.string(),
    answerText: v.string(),
    uploadedEvidenceDocumentId: v.optional(v.string()),
    answeredBy: v.string(),
    legacyAnswerId: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    // Validate case access and authorization
    await validateCaseAccess(ctx, args.caseId, ["POLICYHOLDER", "REVIEWER", "SENIOR_REVIEWER", "ADMIN"], args.answeredBy);

    // 1. Get the question
    const qid = args.questionId;
    const question = await ctx.db
      .query("clarificationQuestions")
      .filter((q) => q.eq(q.field("caseId"), args.caseId))
      .collect();
    
    const targetQuestion = question.find((q) => q._id === qid || q.legacyId === qid);
    if (!targetQuestion) {
      throw new Error("Question not found for this case");
    }

    // 2. Insert answer
    const answerId = await ctx.db.insert("clarificationAnswers", {
      questionId: targetQuestion._id,
      answerText: args.answerText,
      uploadedEvidenceDocumentId: args.uploadedEvidenceDocumentId,
      answeredBy: args.answeredBy,
      legacyId: args.legacyAnswerId,
      createdAt: new Date().toISOString(),
    });

    // 3. Mark question as resolved
    await ctx.db.patch(targetQuestion._id, { isResolved: true });

    // 4. Check if all questions are resolved
    const allQuestions = await ctx.db.query("clarificationQuestions")
      .withIndex("by_case_id", (q) => q.eq("caseId", args.caseId))
      .collect();

    const unresolvedCount = allQuestions.filter((q) => !q.isResolved).length;
    const allResolved = unresolvedCount === 0;

    if (allResolved) {
      // Find case
      const caseItem = await ctx.db.query("cases")
        .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.caseId))
        .first();

      if (caseItem && caseItem.workflowState === "NEEDS_CLARIFICATION") {
        const now = new Date().toISOString();
        await ctx.db.patch(caseItem._id, {
          workflowState: "ANALYSIS_READY",
          updatedAt: now,
        });

        // Insert case status history
        await ctx.db.insert("caseStatusHistory", {
          caseId: caseItem.legacyId || caseItem._id,
          fromState: "NEEDS_CLARIFICATION",
          toState: "ANALYSIS_READY",
          changedBy: args.answeredBy,
          reason: "All clarifying questions resolved by user",
          createdAt: now,
        });
      }
    }

    return {
      answerId,
      allQuestionsResolved: allResolved,
    };
  },
});
