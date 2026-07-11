/**
 * IBM Bob's newer VS Code extension bundle (app/extensions/bob-code, distinct
 * from the older standalone bob-code extension) uses a genuine loopback-style
 * OAuth flow: browser to bob.ibm.com/login?callback_uri=&state=, then a
 * server-side POST /v1/auth/token exchange with a bare {code} body — no
 * client_id/secret, no PKCE. This is unlike the older extension's vscode://
 * -only callback, which strictly rejects any other redirect_uri (confirmed
 * live: non-vscode redirect_uri -> 400 "Invalid redirect_uri").
 *
 * ibm-bob is recategorized OAuth-primary, matching Antigravity's pure-OAuth
 * UI treatment exactly: it is deliberately NOT in FREE_APIKEY_PROVIDER_IDS,
 * so the provider detail page's primary button reads "Add Connection" (not
 * "Add PAT") and opens the browser sign-in flow directly instead of the
 * manual API-key modal. See ProviderDetailPageClient.tsx's `isOAuth =
 * providerSupportsOAuth && !providerSupportsPat` — membership in
 * FREE_APIKEY_PROVIDER_IDS is what flips a provider away from that button.
 */
import test from "node:test";
import assert from "node:assert/strict";

import { AI_PROVIDERS, FREE_APIKEY_PROVIDER_IDS } from "../../src/shared/constants/providers.ts";
import { REGISTRY } from "../../open-sse/config/providerRegistry.ts";
import { supportsTokenRefresh } from "../../open-sse/services/tokenRefresh.ts";
import PROVIDERS_MAP from "../../src/lib/oauth/providers/index.ts";
import { IBM_BOB_CONFIG } from "../../src/lib/oauth/constants/oauth.ts";

test("ibm-bob is registered as an OAuth-primary provider in the UI catalog", () => {
  const p = AI_PROVIDERS["ibm-bob"];
  assert.ok(p, "AI_PROVIDERS['ibm-bob'] must exist");
  assert.equal(p.id, "ibm-bob");
  assert.equal(p.name, "IBM Bob");
});

test("ibm-bob is NOT in FREE_APIKEY_PROVIDER_IDS, so its primary CTA is the OAuth 'Add Connection' button (like Antigravity), not 'Add PAT'", () => {
  assert.equal(FREE_APIKEY_PROVIDER_IDS.has("ibm-bob"), false);
});

test("ibm-bob registry entry keeps its chat-completions shape and gains oauth endpoints", () => {
  const r = REGISTRY["ibm-bob"];
  assert.ok(r, "REGISTRY['ibm-bob'] must exist");
  assert.equal(r.baseUrl, "https://api.us-east.bob.ibm.com/v1/chat/completions");
  assert.equal(r.authHeader, "bearer");
  assert.equal(r.oauth?.tokenUrl, "https://api.us-east.bob.ibm.com/v1/auth/token");
  assert.equal(r.oauth?.refreshUrl, "https://api.us-east.bob.ibm.com/v1/auth/refresh");
});

test("ibm-bob OAuth provider builds the bob.ibm.com/login authorize URL", () => {
  const map = PROVIDERS_MAP as Record<string, any>;
  const provider = map["ibm-bob"];
  assert.ok(provider, "PROVIDERS map must include 'ibm-bob'");
  assert.equal(provider.flowType, "authorization_code");

  const url = provider.buildAuthUrl(IBM_BOB_CONFIG, "http://127.0.0.1:20128/callback", "state-123");
  const parsed = new URL(url);
  assert.equal(parsed.origin + parsed.pathname, "https://bob.ibm.com/login");
  assert.equal(parsed.searchParams.get("callback_uri"), "http://127.0.0.1:20128/callback");
  assert.equal(parsed.searchParams.get("state"), "state-123");
});

test("ibm-bob exchangeToken POSTs {code, callback_uri} with the User-Agent header", async () => {
  const map = PROVIDERS_MAP as Record<string, any>;
  const provider = map["ibm-bob"];
  const origFetch = globalThis.fetch;
  const calls: { url: string; init?: RequestInit }[] = [];
  globalThis.fetch = (async (url: any, init?: RequestInit) => {
    calls.push({ url: String(url), init });
    return new Response(
      JSON.stringify({
        token: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyIjoiYWxpY2VAaWJtLmNvbSIsImV4cCI6OTk5OTk5OTk5OX0.sig",
        refresh_token: "refresh-abc",
        idp_access_token: "idp-at",
        idp_id_token: "idp-it",
      }),
      { status: 200, headers: { "Content-Type": "application/json" } }
    ) as unknown as Response;
  }) as typeof fetch;

  try {
    const redirectUri = "http://127.0.0.1:20128/callback";
    const tokens = await provider.exchangeToken(IBM_BOB_CONFIG, "auth-code-xyz", redirectUri);
    assert.equal(calls.length, 1);
    assert.equal(calls[0].url, "https://api.us-east.bob.ibm.com/v1/auth/token");
    assert.equal(calls[0].init?.method, "POST");
    const body = JSON.parse(calls[0].init?.body as string);
    // Regression guard: the gateway 401s when callback_uri is missing — it validates
    // the redirect URI against the one from the authorization request (RFC 6749 §4.1.3).
    assert.deepEqual(body, { code: "auth-code-xyz", callback_uri: redirectUri });
    assert.equal(tokens.token.startsWith("eyJ"), true);
    // Regression guard: the gateway 401s with {"message":"Authentication
    // required","error":"unauthorized"} when this header is missing (confirmed
    // live) — it is not optional decoration, it's how the client authenticates
    // to this code-only, no-secret token endpoint.
    const headers = new Headers(calls[0].init?.headers);
    assert.equal(headers.get("User-Agent"), IBM_BOB_CONFIG.userAgent);
  } finally {
    globalThis.fetch = origFetch;
  }
});

test("ibm-bob mapTokens decodes the JWT for email/expiry and carries idp tokens", () => {
  const map = PROVIDERS_MAP as Record<string, any>;
  const provider = map["ibm-bob"];
  const exp = Math.floor(Date.now() / 1000) + 3600;
  const payload = Buffer.from(JSON.stringify({ user: "alice@ibm.com", exp })).toString("base64url");
  const jwt = `eyJhbGciOiJIUzI1NiJ9.${payload}.sig`;

  const mapped = provider.mapTokens({
    token: jwt,
    refresh_token: "refresh-abc",
    idp_access_token: "idp-at",
    idp_id_token: "idp-it",
  });

  assert.equal(mapped.accessToken, jwt);
  assert.equal(mapped.refreshToken, "refresh-abc");
  assert.equal(mapped.email, "alice@ibm.com");
  assert.ok(mapped.expiresIn > 3500 && mapped.expiresIn <= 3600);
  assert.equal(mapped.providerSpecificData.idpAccessToken, "idp-at");
  assert.equal(mapped.providerSpecificData.idpIdToken, "idp-it");
});

test("ibm-bob token refresh handler is wired in tokenRefresh.ts", () => {
  assert.equal(supportsTokenRefresh("ibm-bob"), true);
});

test("refreshIbmBobToken POSTs {refresh_token} and returns the new token", async () => {
  const { refreshIbmBobToken } = await import("../../open-sse/services/tokenRefresh.ts");
  const origFetch = globalThis.fetch;
  const calls: { url: string; init?: RequestInit }[] = [];
  globalThis.fetch = (async (url: any, init?: RequestInit) => {
    calls.push({ url: String(url), init });
    return new Response(JSON.stringify({ token: "new-jwt", refresh_token: "new-refresh" }), {
      status: 200,
    }) as unknown as Response;
  }) as typeof fetch;

  try {
    const result = await refreshIbmBobToken("old-refresh", undefined, null);
    assert.equal(calls[0].url, "https://api.us-east.bob.ibm.com/v1/auth/refresh");
    const body = JSON.parse(calls[0].init?.body as string);
    assert.deepEqual(body, { refresh_token: "old-refresh" });
    assert.equal(result?.accessToken, "new-jwt");
    assert.equal(result?.refreshToken, "new-refresh");
    const headers = new Headers(calls[0].init?.headers);
    assert.equal(headers.get("User-Agent"), IBM_BOB_CONFIG.userAgent);
  } finally {
    globalThis.fetch = origFetch;
  }
});

test("refreshIbmBobToken returns the unrecoverable sentinel on 401", async () => {
  const { refreshIbmBobToken } = await import("../../open-sse/services/tokenRefresh.ts");
  const origFetch = globalThis.fetch;
  globalThis.fetch = (async () =>
    new Response(JSON.stringify({ error: "invalid_grant" }), { status: 401 })) as typeof fetch;

  try {
    const result = await refreshIbmBobToken("dead-refresh", undefined, null);
    assert.equal(result?.error, "unrecoverable_refresh_error");
  } finally {
    globalThis.fetch = origFetch;
  }
});
