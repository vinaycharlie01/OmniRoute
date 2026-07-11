/**
 * Parsing/error-message logic for OAuthModal's manual "paste the callback URL"
 * step. A user reported "No authorization code found" after pasting IBM Bob's
 * sign-in link (https://bob.ibm.com/login?callback_uri=&state=) into that
 * field — the sign-in URL never carries a `code` param by definition, since
 * it's step 1 (the link you open), not step 2 (the page you land on after
 * completing login). buildNoAuthCodeErrorMessage detects that specific
 * mistake and gives an actionable hint instead of the generic message.
 */
import test from "node:test";
import assert from "node:assert/strict";
import {
  parseManualOAuthCallback,
  buildNoAuthCodeErrorMessage,
} from "../../src/shared/components/oauthManualCallbackParse.ts";

test("parseManualOAuthCallback extracts code/state/error from a full callback URL", () => {
  const parsed = parseManualOAuthCallback(
    "https://omnir-production-91cb.up.railway.app/callback?code=abc123&state=xyz",
    null
  );
  assert.equal(parsed.code, "abc123");
  assert.equal(parsed.state, "xyz");
  assert.equal(parsed.error, null);
});

test("parseManualOAuthCallback falls back to the flow's state when the URL carries none", () => {
  const parsed = parseManualOAuthCallback(
    "https://omnir-production-91cb.up.railway.app/callback?code=abc123",
    "fallback-state"
  );
  assert.equal(parsed.code, "abc123");
  assert.equal(parsed.state, "fallback-state");
});

test("parseManualOAuthCallback surfaces error/error_description params", () => {
  const parsed = parseManualOAuthCallback(
    "https://host/callback?error=access_denied&error_description=User+cancelled",
    null
  );
  assert.equal(parsed.error, "access_denied");
  assert.equal(parsed.errorDescription, "User cancelled");
});

test("parseManualOAuthCallback parses a raw code#state fragment (Claude Code / Cline)", () => {
  const parsed = parseManualOAuthCallback("abc123#xyz", null);
  assert.equal(parsed.code, "abc123");
  assert.equal(parsed.state, "xyz");
});

test("parseManualOAuthCallback returns no code for a bare sign-in URL (no code param)", () => {
  const parsed = parseManualOAuthCallback(
    "https://bob.ibm.com/login?callback_uri=https%3A%2F%2Fomnir-production-91cb.up.railway.app%2Fcallback&state=oCIAKPr2YZR5jY-P9BF8lKYXENJ-ceuG3hJrFJC_dUI",
    null
  );
  assert.equal(parsed.code, null);
});

test("buildNoAuthCodeErrorMessage detects the pasted sign-in URL mistake and names the expected origin", () => {
  const msg = buildNoAuthCodeErrorMessage(
    "https://bob.ibm.com/login?callback_uri=https%3A%2F%2Fomnir-production-91cb.up.railway.app%2Fcallback&state=oCIAKPr2YZR5jY-P9BF8lKYXENJ-ceuG3hJrFJC_dUI",
    "https://omnir-production-91cb.up.railway.app/callback"
  );
  assert.match(msg, /sign-in link/);
  assert.match(msg, /https:\/\/omnir-production-91cb\.up\.railway\.app/);
});

test("buildNoAuthCodeErrorMessage falls back to the generic message when the origin matches", () => {
  const msg = buildNoAuthCodeErrorMessage(
    "https://omnir-production-91cb.up.railway.app/callback?state=xyz",
    "https://omnir-production-91cb.up.railway.app/callback"
  );
  assert.equal(
    msg,
    "No authorization code found. Paste the callback URL or the Authentication Code."
  );
});

test("buildNoAuthCodeErrorMessage falls back to the generic message for non-URL input", () => {
  const msg = buildNoAuthCodeErrorMessage(
    "",
    "https://omnir-production-91cb.up.railway.app/callback"
  );
  assert.equal(
    msg,
    "No authorization code found. Paste the callback URL or the Authentication Code."
  );
});

test("buildNoAuthCodeErrorMessage falls back to the generic message when redirectUri is unknown", () => {
  const msg = buildNoAuthCodeErrorMessage("https://bob.ibm.com/login?state=xyz", null);
  assert.equal(
    msg,
    "No authorization code found. Paste the callback URL or the Authentication Code."
  );
});
