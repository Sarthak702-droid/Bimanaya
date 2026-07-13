// Clerk authentication configuration for Convex.
// Clerk-issued JWTs are validated against this issuer domain.
// The CLERK_JWT_ISSUER_DOMAIN environment variable must be set in the
// Convex dashboard (e.g. "https://alert-ghost-7.clerk.accounts.dev").
export default {
  providers: [
    {
      domain: process.env.CLERK_JWT_ISSUER_DOMAIN!,
      applicationID: "convex",
    },
  ],
};
