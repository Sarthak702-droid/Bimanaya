import { query, mutation } from "./_generated/server";
import { v } from "convex/values";

// ── User Sync (called after Clerk login) ────────────────────────────────
export const syncCurrentUser = mutation({
  args: {
    clerkUserId: v.string(),
    clerkSubject: v.string(),
    email: v.string(),
    emailVerified: v.boolean(),
    firstName: v.optional(v.string()),
    lastName: v.optional(v.string()),
    displayName: v.optional(v.string()),
    imageUrl: v.optional(v.string()),
    phone: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const identity = await ctx.auth.getUserIdentity();
    if (!identity) {
      throw new Error("Unauthenticated: cannot sync user");
    }

    // Look for existing user by Clerk user ID
    const existing = await ctx.db
      .query("users")
      .withIndex("by_clerk_user_id", (q) => q.eq("clerkUserId", args.clerkUserId))
      .first();

    const now = new Date().toISOString();

    if (existing) {
      // Update safe profile fields — never overwrite role or status from frontend
      await ctx.db.patch(existing._id, {
        email: args.email,
        emailVerified: args.emailVerified,
        firstName: args.firstName,
        lastName: args.lastName,
        displayName: args.displayName,
        imageUrl: args.imageUrl,
        phone: args.phone,
        lastLoginAt: now,
        updatedAt: now,
      });
      return existing._id;
    }

    // Create new user profile
    const userId = await ctx.db.insert("users", {
      clerkUserId: args.clerkUserId,
      clerkSubject: args.clerkSubject,
      email: args.email,
      emailVerified: args.emailVerified,
      firstName: args.firstName,
      lastName: args.lastName,
      displayName: args.displayName,
      imageUrl: args.imageUrl,
      phone: args.phone,
      preferredLanguage: "en",
      role: "POLICYHOLDER",
      status: "ACTIVE",
      onboardingCompleted: false,
      lastLoginAt: now,
      createdAt: now,
      updatedAt: now,
    });

    return userId;
  },
});

// ── Get Current User ────────────────────────────────────────────────────
export const getCurrent = query({
  args: {},
  handler: async (ctx) => {
    const identity = await ctx.auth.getUserIdentity();
    if (!identity) {
      return null;
    }

    const user = await ctx.db
      .query("users")
      .withIndex("by_clerk_subject", (q) => q.eq("clerkSubject", identity.subject))
      .first();

    return user;
  },
});

// ── Get User by Clerk ID ────────────────────────────────────────────────
export const getByClerkId = query({
  args: { clerkUserId: v.string() },
  handler: async (ctx, args) => {
    const identity = await ctx.auth.getUserIdentity();
    if (!identity) {
      throw new Error("Unauthenticated");
    }

    return await ctx.db
      .query("users")
      .withIndex("by_clerk_user_id", (q) => q.eq("clerkUserId", args.clerkUserId))
      .first();
  },
});

// ── Get User by Email ───────────────────────────────────────────────────
export const getByEmail = query({
  args: { email: v.string() },
  handler: async (ctx, args) => {
    const identity = await ctx.auth.getUserIdentity();
    if (!identity) {
      throw new Error("Unauthenticated");
    }

    return await ctx.db
      .query("users")
      .withIndex("by_email", (q) => q.eq("email", args.email))
      .first();
  },
});

// ── Get User by Legacy ID (migration) ───────────────────────────────────
export const getByLegacyId = query({
  args: { legacyId: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db
      .query("users")
      .withIndex("by_legacy_id", (q) => q.eq("legacyId", args.legacyId))
      .first();
  },
});

// ── Update User Role (admin only, called from Go backend) ───────────────
export const updateRole = mutation({
  args: {
    clerkUserId: v.string(),
    role: v.union(
      v.literal("POLICYHOLDER"),
      v.literal("FAMILY_MEMBER"),
      v.literal("REVIEWER"),
      v.literal("SENIOR_REVIEWER"),
      v.literal("PARTNER"),
      v.literal("ADMIN"),
      v.literal("OPERATIONS")
    ),
  },
  handler: async (ctx, args) => {
    const user = await ctx.db
      .query("users")
      .withIndex("by_clerk_user_id", (q) => q.eq("clerkUserId", args.clerkUserId))
      .first();

    if (!user) {
      throw new Error("User not found");
    }

    await ctx.db.patch(user._id, {
      role: args.role,
      updatedAt: new Date().toISOString(),
    });

    return user._id;
  },
});

// ── Get User by Email S2S (called from Go API auth handler) ──────────────
export const getByEmailS2S = query({
  args: { email: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db
      .query("users")
      .withIndex("by_email", (q) => q.eq("email", args.email))
      .first();
  },
});

// ── Register User S2S (called from Go API auth handler) ──────────────────
export const registerUserS2S = mutation({
  args: {
    email: v.string(),
    phone: v.optional(v.string()),
    role: v.string(),
    legacyId: v.string(),
  },
  handler: async (ctx, args) => {
    const now = new Date().toISOString();
    return await ctx.db.insert("users", {
      clerkUserId: args.legacyId,
      clerkSubject: args.legacyId,
      email: args.email,
      emailVerified: true,
      phone: args.phone,
      preferredLanguage: "en",
      role: args.role as any,
      status: "ACTIVE",
      onboardingCompleted: false,
      lastLoginAt: now,
      createdAt: now,
      updatedAt: now,
      legacyId: args.legacyId,
    });
  },
});
