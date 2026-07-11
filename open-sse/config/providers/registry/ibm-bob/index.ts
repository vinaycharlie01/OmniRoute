import type { RegistryEntry } from "../../shared.ts";

/**
 * IBM Bob (Bob Code) — the LiteLLM-compatible enterprise gateway backing the
 * IBM Bob VS Code extension (github.com/IBM/Bob). The extension itself signs
 * in via IBM's own SSO flow and receives a short-lived Bearer token; there is
 * no public OAuth client to embed, so users paste their own Bob access token
 * here as a plain Bearer API key, same as any other apikey OpenAI-compatible
 * provider. Bob's client defaults to the "premium" model alias, routed
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
};
