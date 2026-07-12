/**
 * IBM Bob's gateway only ever serves POST /chat/completions with the "premium"
 * model alias (confirmed from the Bob VS Code extension's own client, which
 * never calls GET /models). The generic validateOpenAILikeProvider path probes
 * GET /models first and would short-circuit to "Invalid API key" on any 401/403
 * there — a false negative if that route isn't authorized for chat-scoped
 * tokens even when the token is genuinely valid for chat completions. bob
 * has its own dedicated validator (bypasses /models entirely) instead of
 * falling through to the generic default.
 *
 * The endpoint (/inference/v1, not /v1) and header (x-api-key, not
 * Authorization: Bearer) are confirmed against a working published reference
 * client (github.com/Kynareth01/bob-proxy) after the OAuth token exchange
 * proved unreachable in practice.
 */
import test from "node:test";
import assert from "node:assert/strict";

const { validateProviderApiKey } = await import("../../src/lib/providers/validation.ts");

test("bob validates via a single POST /chat/completions call (no /models probe), using x-api-key", async () => {
  const originalFetch = globalThis.fetch;
  const calls: { url: string; method: string; headers: Headers; body: unknown }[] = [];
  globalThis.fetch = (async (url: string, init: RequestInit = {}) => {
    calls.push({
      url: String(url),
      method: init.method || "GET",
      headers: new Headers(init.headers),
      body: init.body ? JSON.parse(init.body as string) : null,
    });
    return new Response(JSON.stringify({ choices: [{ message: { content: "ok" } }] }), {
      status: 200,
    });
  }) as typeof fetch;

  try {
    const result = await validateProviderApiKey({
      provider: "bob",
      apiKey: "valid-bob-token",
      providerSpecificData: {},
    });
    assert.equal(result.valid, true);
    assert.equal(calls.length, 1, "must not probe GET /models before the chat completions call");
    assert.equal(calls[0].method, "POST");
    assert.equal(calls[0].url, "https://api.us-east.bob.ibm.com/inference/v1/chat/completions");
    assert.equal((calls[0].body as { model: string }).model, "premium");
    assert.equal(calls[0].headers.get("x-api-key"), "valid-bob-token");
    assert.equal(calls[0].headers.get("Authorization"), null);
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test("bob reports Invalid API key on a real 401 from /chat/completions", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = (async () =>
    new Response(JSON.stringify({ error: "invalid token" }), { status: 401 })) as typeof fetch;

  try {
    const result = await validateProviderApiKey({
      provider: "bob",
      apiKey: "bad-token",
      providerSpecificData: {},
    });
    assert.equal(result.valid, false);
    assert.equal(result.error, "Invalid API key");
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test("bob honors a region-overridden base URL", async () => {
  const originalFetch = globalThis.fetch;
  let requestedUrl = "";
  globalThis.fetch = (async (url: string) => {
    requestedUrl = String(url);
    return new Response(JSON.stringify({ choices: [{ message: { content: "ok" } }] }), {
      status: 200,
    });
  }) as typeof fetch;

  try {
    const result = await validateProviderApiKey({
      provider: "bob",
      apiKey: "valid-bob-token",
      providerSpecificData: { baseUrl: "https://api.eu-west.bob.ibm.com/inference/v1" },
    });
    assert.equal(result.valid, true);
    assert.equal(requestedUrl, "https://api.eu-west.bob.ibm.com/inference/v1/chat/completions");
  } finally {
    globalThis.fetch = originalFetch;
  }
});
