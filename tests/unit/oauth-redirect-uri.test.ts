/**
 * Redirect-URI resolution for OAuthModal's "start login" step.
 *
 * The user reported IBM Bob's OAuth flow producing a public HTTPS
 * callback_uri (their Railway domain) that IBM's server didn't seem to
 * complete cleanly. IBM Bob's own VS Code extension client
 * (app/extensions/bob-code) always hardcodes a loopback callback_uri
 * (http://127.0.0.1:<port>) — it never tries a public domain — so
 * bob.ibm.com's acceptance of a public HTTPS redirect_uri was never actually
 * verified. ibm-bob now gets the same loopback-always treatment as the
 * Google OAuth providers (antigravity/agy/gemini-cli), which already ship
 * this exact pattern for the same underlying reason (their embedded OAuth
 * client only documents loopback support).
 */
import test from "node:test";
import assert from "node:assert/strict";
import {
  resolveOAuthModalRedirectUri,
  LOOPBACK_ONLY_OAUTH_PROVIDERS,
  GOOGLE_OAUTH_PROVIDERS,
} from "../../src/shared/components/oauthRedirectUri.ts";

const remoteCtx = {
  origin: "https://omnir-production-91cb.up.railway.app",
  port: "",
  protocol: "https:",
  isLocalhost: false,
  publicBaseUrl: null,
};

const localCtx = {
  origin: "http://localhost:20128",
  port: "20128",
  protocol: "http:",
  isLocalhost: true,
  publicBaseUrl: null,
};

test("ibm-bob is in LOOPBACK_ONLY_OAUTH_PROVIDERS alongside the Google OAuth providers", () => {
  assert.ok(LOOPBACK_ONLY_OAUTH_PROVIDERS.has("ibm-bob"));
  for (const p of GOOGLE_OAUTH_PROVIDERS) {
    assert.ok(LOOPBACK_ONLY_OAUTH_PROVIDERS.has(p));
  }
});

test("ibm-bob always resolves to a loopback redirect_uri, even on a remote (Railway) dashboard", () => {
  const uri = resolveOAuthModalRedirectUri("ibm-bob", remoteCtx);
  assert.equal(uri, "http://127.0.0.1:20128/callback");
  assert.equal(uri.includes("railway"), false);
});

test("ibm-bob uses the dashboard's own port for the loopback redirect_uri", () => {
  const uri = resolveOAuthModalRedirectUri("ibm-bob", { ...remoteCtx, port: "8080" });
  assert.equal(uri, "http://127.0.0.1:8080/callback");
});

test("antigravity (Google OAuth) still resolves to loopback on remote, matching ibm-bob", () => {
  const uri = resolveOAuthModalRedirectUri("antigravity", remoteCtx);
  assert.equal(uri, "http://127.0.0.1:20128/callback");
});

test("a generic remote provider (e.g. gitlab-duo) uses the actual origin, not loopback", () => {
  const uri = resolveOAuthModalRedirectUri("gitlab-duo", remoteCtx);
  assert.equal(uri, "https://omnir-production-91cb.up.railway.app/callback");
});

test("a generic remote provider honors NEXT_PUBLIC_BASE_URL override", () => {
  const uri = resolveOAuthModalRedirectUri("gitlab-duo", {
    ...remoteCtx,
    publicBaseUrl: "https://omniroute.example.com",
  });
  assert.equal(uri, "https://omniroute.example.com/callback");
});

test("codex/openai always use the fixed loopback callback port 1455", () => {
  assert.equal(
    resolveOAuthModalRedirectUri("codex", remoteCtx),
    "http://localhost:1455/auth/callback"
  );
  assert.equal(
    resolveOAuthModalRedirectUri("openai", localCtx),
    "http://localhost:1455/auth/callback"
  );
});

test("windsurf/devin-cli use localhost + /auth/callback on remote", () => {
  assert.equal(
    resolveOAuthModalRedirectUri("windsurf", remoteCtx),
    "http://localhost:20128/auth/callback"
  );
  assert.equal(
    resolveOAuthModalRedirectUri("devin-cli", { ...remoteCtx, port: "8080" }),
    "http://localhost:8080/auth/callback"
  );
});

test("a generic provider on true localhost uses localhost:port", () => {
  const uri = resolveOAuthModalRedirectUri("gitlab-duo", localCtx);
  assert.equal(uri, "http://localhost:20128/callback");
});
