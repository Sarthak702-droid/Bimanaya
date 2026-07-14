import { query, mutation } from "./_generated/server";
import { v } from "convex/values";
import { requireRole } from "./authHelpers";

// Helper to get current authenticated user or fallback in development
export async function getCurrentUserOrThrow(ctx: any, fallbackUserId?: string) {
  const identity = await ctx.auth.getUserIdentity();
  
  if (!identity) {
    // Fallback for development environment only
    if (process.env.ENV === "development" && fallbackUserId) {
      let devUser = await ctx.db
        .query("users")
        .withIndex("by_legacy_id", (q: any) => q.eq("legacyId", fallbackUserId))
        .first();
      
      if (!devUser) {
        devUser = await ctx.db
          .query("users")
          .withIndex("by_clerk_user_id", (q: any) => q.eq("clerkUserId", fallbackUserId))
          .first();
      }

      if (devUser) {
        if (devUser.status !== "ACTIVE") {
          throw new Error("USER_INACTIVE");
        }
        return devUser;
      }
    }
    throw new Error("UNAUTHENTICATED");
  }

  const user = await ctx.db
    .query("users")
    .withIndex("by_clerk_subject", (q: any) => q.eq("clerkSubject", identity.subject))
    .first();

  if (!user) {
    throw new Error("USER_NOT_FOUND");
  }
  
  if (user.status !== "ACTIVE") {
    throw new Error("USER_INACTIVE");
  }
  return user;
}

// ── User Sync (called after Clerk login) ────────────────────────────────
export const syncCurrentUser = mutation({
  args: {
    clerkUserId: v.optional(v.string()),
    clerkSubject: v.optional(v.string()),
    email: v.optional(v.string()),
    emailVerified: v.optional(v.boolean()),
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

    // Derive critical fields strictly from Clerk identity
    const clerkUserId = identity.subject;
    const clerkSubject = identity.subject;
    const email = identity.email || args.email || "";
    const emailVerified = identity.emailVerified ?? args.emailVerified ?? false;

    if (!email) {
      throw new Error("Email not found in identity or arguments");
    }

    // Look for existing user by Clerk user ID
    const existing = await ctx.db
      .query("users")
      .withIndex("by_clerk_user_id", (q) => q.eq("clerkUserId", clerkUserId))
      .first();

    const now = new Date().toISOString();

    if (existing) {
      // Update safe profile fields — never overwrite role or status from frontend
      await ctx.db.patch(existing._id, {
        email,
        emailVerified,
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
      clerkUserId,
      clerkSubject,
      email,
      emailVerified,
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
    // Restrict this to authenticated users
    const identity = await ctx.auth.getUserIdentity();
    if (!identity && process.env.ENV !== "development") {
      throw new Error("Unauthenticated");
    }

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
    // Only allow ADMIN to update user roles
    const admin = await requireRole(ctx, ["ADMIN"]);

    const user = await ctx.db
      .query("users")
      .withIndex("by_clerk_user_id", (q) => q.eq("clerkUserId", args.clerkUserId))
      .first();

    if (!user) {
      throw new Error("User not found");
    }

    // Prevent self-role assignment
    if (admin._id === user._id) {
      throw new Error("FORBIDDEN: Admins cannot change their own roles");
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
    // Restrict to development environment only
    if (process.env.ENV !== "development") {
      throw new Error("UNAUTHORIZED: S2S endpoint only available in development");
    }

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
    // Restrict to development environment only
    if (process.env.ENV !== "development") {
      throw new Error("UNAUTHORIZED: S2S endpoint only available in development");
    }

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
