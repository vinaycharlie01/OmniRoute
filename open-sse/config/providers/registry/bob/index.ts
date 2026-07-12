import type { RegistryEntry } from "../../shared.ts";

/**
 * Bob (formerly registered as "ibm-bob") — the LiteLLM-compatible enterprise
 * gateway backing the IBM Bob VS Code extension (github.com/IBM/Bob).
 * API-key-primary: a standalone Bob API key is sent as `x-api-key` against
 * the gateway's `/inference/v1` service path, confirmed against a working
 * published reference client (github.com/Kynareth01/bob-proxy) after the
 * OAuth authorization_code flow (bob.ibm.com/login, see
 * src/lib/oauth/providers/bob.ts) proved unreachable in practice — IBM's
 * `/v1/auth/token` endpoint kept returning "Authentication required" even
 * with a byte-for-byte correct request. OAuth code stays wired for a future
 * fix but is no longer the default UI flow (see FREE_APIKEY_PROVIDER_IDS in
 * src/shared/constants/providers.ts).
 *
 * The model list below was read directly from a live `GET
 * /inference/v1/model/info` call (confirmed working with a real key) — see
 * `PROVIDER_MODELS_CONFIG.bob` in src/app/api/providers/[id]/models/route.ts
 * for the live-discovery wiring that keeps this list fresh going forward.
 * A few gateway-internal/utility entries observed in that response
 * (`greetings`, `ownership`, `identity`, `explorer`, `rnj-1-*`, region-locked
 * `-global-ibm-only` variants) are intentionally left out of this curated
 * default list — passthroughModels still allows using them explicitly.
 */
export const bobProvider: RegistryEntry = {
  id: "bob",
  alias: "bob",
  format: "openai",
  executor: "default",
  baseUrl: "https://api.us-east.bob.ibm.com/inference/v1/chat/completions",
  authType: "apikey",
  authHeader: "x-api-key",
  models: [
    { id: "premium", name: "Bob Premium (default alias)" },
    { id: "premium-ide", name: "Bob Premium (IDE)" },
    { id: "premium-shell", name: "Bob Premium (Shell)" },
    { id: "sonnet-4.5", name: "Claude Sonnet 4.5" },
    { id: "sonnet-4.5-us-west", name: "Claude Sonnet 4.5 (US West)" },
    { id: "sonnet-4.6", name: "Claude Sonnet 4.6" },
    { id: "sonnet-4.6-us-west", name: "Claude Sonnet 4.6 (US West)" },
    { id: "haiku-4.5", name: "Claude Haiku 4.5" },
    { id: "gpt-2026-5.4", name: "GPT 2026-5.4" },
    { id: "gemini-3.5-flash", name: "Gemini 3.5 Flash" },
    { id: "granite-8b-code-instruct", name: "Granite 8B Code Instruct" },
    { id: "gpt-oss-20b", name: "GPT-OSS 20B" },
    { id: "mistral-medium-3.1", name: "Mistral Medium 3.1" },
  ],
  passthroughModels: true,
  oauth: {
    tokenUrl: "https://api.us-east.bob.ibm.com/v1/auth/token",
    refreshUrl: "https://api.us-east.bob.ibm.com/v1/auth/refresh",
  },
};
