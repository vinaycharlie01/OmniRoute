import type { RegistryEntry } from "../../shared.ts";

/**
 * IBM Bob (Bob Code) — the LiteLLM-compatible enterprise gateway backing the
 * IBM Bob VS Code extension (github.com/IBM/Bob). API-key-primary: a
 * standalone Bob API key is sent as `x-api-key` against the gateway's
 * `/inference/v1` service path, confirmed against a working published
 * reference client (github.com/Kynareth01/bob-proxy) after the OAuth
 * authorization_code flow (bob.ibm.com/login, see
 * src/lib/oauth/providers/ibm-bob.ts) proved unreachable in practice — IBM's
 * `/v1/auth/token` endpoint kept returning "Authentication required" even
 * with a byte-for-byte correct request. OAuth code stays wired for a future
 * fix but is no longer the default UI flow (see FREE_APIKEY_PROVIDER_IDS in
 * src/shared/constants/providers.ts). Bob's client defaults to the
 * "premium" model alias, routed server-side by IBM's gateway.
 */
export const ibm_bobProvider: RegistryEntry = {
  id: "ibm-bob",
  alias: "ibm-bob",
  format: "openai",
  executor: "default",
  baseUrl: "https://api.us-east.bob.ibm.com/inference/v1/chat/completions",
  authType: "apikey",
  authHeader: "x-api-key",
  models: [{ id: "premium", name: "Bob Premium (default alias)" }],
  passthroughModels: true,
  oauth: {
    tokenUrl: "https://api.us-east.bob.ibm.com/v1/auth/token",
    refreshUrl: "https://api.us-east.bob.ibm.com/v1/auth/refresh",
  },
};
