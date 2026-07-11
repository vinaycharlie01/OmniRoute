import type { RegistryEntry } from "../../shared.ts";

/**
 * IBM Bob (Bob Code) — the LiteLLM-compatible enterprise gateway backing the
 * IBM Bob VS Code extension (github.com/IBM/Bob). Two ways to connect:
 *  - OAuth: sign in via bob.ibm.com/login (authorization_code, no client
 *    secret) — see src/lib/oauth/providers/ibm-bob.ts. This is the primary
 *    flow (matches the newer Bob extension bundle's own client).
 *  - Manual: paste an existing Bob access token as a plain Bearer API key
 *    (kept for users who already have one, or whose deployment can't reach
 *    bob.ibm.com's login page).
 * Both shapes resolve to the same Bearer credential at request time
 * (`credentials.apiKey || credentials.accessToken`), so no executor branching
 * is needed. Bob's client defaults to the "premium" model alias, routed
 * server-side by IBM's gateway.
 */
export const ibm_bobProvider: RegistryEntry = {
  id: "ibm-bob",
  alias: "ibm-bob",
  format: "openai",
  executor: "default",
  baseUrl: "https://api.us-east.bob.ibm.com/v1/chat/completions",
  authType: "apikey",
  authHeader: "bearer",
  models: [{ id: "premium", name: "Bob Premium (default alias)" }],
  passthroughModels: true,
  oauth: {
    tokenUrl: "https://api.us-east.bob.ibm.com/v1/auth/token",
    refreshUrl: "https://api.us-east.bob.ibm.com/v1/auth/refresh",
  },
};
