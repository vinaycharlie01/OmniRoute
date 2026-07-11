/**
 * The OAuth exchange route's catch-all previously returned a flat
 * "Internal server error" for ANY failure in POST /api/oauth/[provider]/exchange
 * (network error, upstream rejection, DB error, etc.) — masking the actual
 * cause from both the client and (via the JSON body, though server logs still
 * had it) anyone debugging a live deployment. A user reported exactly this
 * "Internal server error" wall after IBM Bob's real /v1/auth/token exchange
 * failed. The catch-all now runs the thrown error through sanitizeErrorMessage
 * (already used elsewhere in this same route) instead of a hardcoded string,
 * so the safe (path/stack-stripped) upstream detail reaches the response body.
 */
import test from "node:test";
import assert from "node:assert/strict";

const { POST } = await import("../../src/app/api/oauth/[provider]/[action]/route.ts");

test("POST /api/oauth/ibm-bob/exchange surfaces the sanitized upstream error instead of a flat 'Internal server error'", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = (async (url: string) => {
    if (String(url).includes("api.us-east.bob.ibm.com/v1/auth/token")) {
      return new Response("invalid_grant: authorization code expired", { status: 400 });
    }
    return originalFetch(url as never);
  }) as typeof fetch;

  try {
    const request = new Request("http://localhost:20128/api/oauth/ibm-bob/exchange", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        code: "some-code",
        redirectUri: "http://127.0.0.1:20128/callback",
      }),
    });
    const response = await POST(request, {
      params: Promise.resolve({ provider: "ibm-bob", action: "exchange" }),
    } as never);
    const body = (await response.json()) as { error?: unknown };

    assert.equal(response.status, 500);
    assert.notEqual(body.error, "Internal server error");
    assert.match(String(body.error), /invalid_grant|authorization code expired/);
    // Regression guard for the sanitizer contract: no raw stack/path leakage.
    assert.equal(String(body.error).includes("at /"), false);
  } finally {
    globalThis.fetch = originalFetch;
  }
});
