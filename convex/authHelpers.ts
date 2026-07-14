import { QueryCtx, MutationCtx } from "./_generated/server";
import { Doc, Id } from "./_generated/dataModel";

export async function getCurrentUser(ctx: QueryCtx | MutationCtx) {
  const identity = await ctx.auth.getUserIdentity();
  if (!identity) {
    return null;
  }
  return await ctx.db
    .query("users")
    .withIndex("by_clerk_subject", (q) => q.eq("clerkSubject", identity.subject))
    .first();
}

export async function getCurrentUserOrThrow(
  ctx: any,
  fallbackUserId?: string
): Promise<Doc<"users">> {
  const user = await getCurrentUser(ctx);
  if (user) {
    if (user.status !== "ACTIVE") {
      throw new Error("USER_INACTIVE");
    }
    return user;
  }

  // Fallback for development environment only
  if (process.env.ENV === "development") {
    if (fallbackUserId) {
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

      // Auto-create user in local Convex DB if missing in dev environment
      if (!devUser) {
        const now = new Date().toISOString();
        const mockEmail = fallbackUserId.includes("@") ? fallbackUserId : "test@bimanyaya.in";
        const role = fallbackUserId.includes("reviewer") ? "REVIEWER" : (fallbackUserId.includes("admin") ? "ADMIN" : "POLICYHOLDER");
        
        const newUserId = await ctx.db.insert("users", {
          clerkUserId: fallbackUserId,
          clerkSubject: fallbackUserId,
          email: mockEmail,
          emailVerified: true,
          preferredLanguage: "en",
          role: role,
          status: "ACTIVE",
          onboardingCompleted: true,
          legacyId: fallbackUserId,
          lastLoginAt: now,
          createdAt: now,
          updatedAt: now,
        });
        devUser = await ctx.db.get(newUserId);
      }

      if (devUser) {
        if (devUser.status !== "ACTIVE") {
          throw new Error("USER_INACTIVE");
        }
        return devUser;
      }
    }

    // Default mock user for dev environment if no fallbackUserId is specified
    let firstUser = await ctx.db.query("users").first();
    if (!firstUser) {
      const now = new Date().toISOString();
      const newUserId = await ctx.db.insert("users", {
        clerkUserId: "mock_default",
        clerkSubject: "mock_default",
        email: "test@bimanyaya.in",
        emailVerified: true,
        preferredLanguage: "en",
        role: "POLICYHOLDER",
        status: "ACTIVE",
        onboardingCompleted: true,
        legacyId: "mock_default",
        lastLoginAt: now,
        createdAt: now,
        updatedAt: now,
      });
      firstUser = await ctx.db.get(newUserId);
    }
    if (firstUser) {
      return firstUser;
    }
  }

  throw new Error("UNAUTHENTICATED");
}

export async function requireRole(
  ctx: QueryCtx | MutationCtx,
  allowedRoles: string[],
  fallbackUserId?: string
): Promise<Doc<"users">> {
  const user = await getCurrentUserOrThrow(ctx, fallbackUserId);
  if (!allowedRoles.includes(user.role) && process.env.ENV !== "development") {
    throw new Error("FORBIDDEN");
  }
  return user;
}

export async function validateCaseAccess(
  ctx: QueryCtx | MutationCtx,
  caseId: string,
  allowedRoles: string[] = ["REVIEWER", "SENIOR_REVIEWER", "ADMIN"],
  fallbackUserId?: string
): Promise<{ user: Doc<"users">; caseObj: Doc<"cases"> }> {
  const user = await getCurrentUserOrThrow(ctx, fallbackUserId);

  // Fetch case by legacyId or direct ID
  let caseObj = await ctx.db
    .query("cases")
    .withIndex("by_legacy_id", (q) => q.eq("legacyId", caseId))
    .first();

  if (!caseObj) {
    try {
      caseObj = await ctx.db.get(caseId as any);
    } catch (e) {
      // Not a valid ID format
    }
  }

  if (!caseObj) {
    throw new Error("CASE_NOT_FOUND");
  }

  // Check ownership for POLICYHOLDER
  if (user.role === "POLICYHOLDER") {
    // Check if the user is the owner
    const isOwner =
      caseObj.ownerUserId === user._id ||
      caseObj.ownerUserId === user.legacyId ||
      caseObj.ownerUserId === user.clerkUserId;
    
    if (!isOwner && process.env.ENV !== "development") {
      throw new Error("UNAUTHORIZED_CASE_ACCESS");
    }
  } else {
    // Check role permissions for others (reviewers, admins, etc.)
    if (!allowedRoles.includes(user.role) && process.env.ENV !== "development") {
      throw new Error("FORBIDDEN_CASE_ACCESS");
    }
  }

  return { user, caseObj };
}
