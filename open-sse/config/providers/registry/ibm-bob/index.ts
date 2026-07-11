import type { RegistryEntry } from "../../shared.ts";

/**
 * IBM Bob (Bob Code) — the LiteLLM-compatible enterprise gateway backing the
 * IBM Bob VS Code extension (github.com/IBM/Bob). Connects via OAuth: sign in
 * through bob.ibm.com/login (authorization_code, no client secret) — see
 * src/lib/oauth/providers/ibm-bob.ts — matching the newer Bob extension
 * bundle's own client. `authType: "apikey"` below is a request-shape label,
 * not a UI path: OAuth connections are stored with `accessToken`, and the
 * executor resolves either shape the same way at request time
 * (`credentials.apiKey || credentials.accessToken`), so no branching is
 * needed. Bob's client defaults to the "premium" model alias, routed
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
