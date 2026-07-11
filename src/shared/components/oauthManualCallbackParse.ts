/**
 * Parsing helpers for OAuthModal's manual "paste the callback URL" step.
 * Extracted to keep OAuthModal within its frozen size budget and to make the
 * parsing/error-message logic unit-testable without rendering the component.
 */

export interface ParsedManualOAuthCallback {
  code: string | null;
  state: string | null;
  error: string | null;
  errorDescription: string | null;
}

/**
 * Parse a manually-pasted callback URL (e.g. `https://host/callback?code=&state=`)
 * or a raw "Authentication Code" in `code#state` form (Claude Code / Cline).
 * Falls back to `fallbackState` (the state the flow was started with) when the
 * input carries no state of its own.
 */
export function parseManualOAuthCallback(
  input: string,
  fallbackState: string | null
): ParsedManualOAuthCallback {
  const trimmed = input.trim();
  try {
    const url = new URL(trimmed);
    return {
      code: url.searchParams.get("code"),
      state: url.searchParams.get("state") || url.hash.replace(/^#/, "") || fallbackState,
      error: url.searchParams.get("error"),
      errorDescription: url.searchParams.get("error_description"),
    };
  } catch {
    const [rawCode, rawState] = trimmed.split("#", 2);
    return {
      code: rawCode || null,
      state: rawState || fallbackState,
      error: null,
      errorDescription: null,
    };
  }
}

const GENERIC_NO_CODE_MESSAGE =
  "No authorization code found. Paste the callback URL or the Authentication Code.";

/**
 * Build the error shown when no `code` was found in the pasted input. If the
 * input parses as a URL whose origin doesn't match the expected callback
 * origin, the user most likely pasted the sign-in link they opened (step 1)
 * instead of the page it redirected them to after completing login (step 2) —
 * call that out explicitly instead of the generic message.
 */
export function buildNoAuthCodeErrorMessage(input: string, redirectUri?: string | null): string {
  if (!redirectUri) return GENERIC_NO_CODE_MESSAGE;
  try {
    const pasted = new URL(input.trim());
    const expected = new URL(redirectUri);
    if (pasted.origin !== expected.origin) {
      return (
        "That looks like the sign-in link you opened, not the page it redirected you to after " +
        "completing login. Finish signing in first, then copy the final URL from your browser's " +
        `address bar (it should start with ${expected.origin}) and paste that here instead.`
      );
    }
  } catch {
    // Not a URL at all (blank input, or a raw code fragment) — keep the generic message.
  }
  return GENERIC_NO_CODE_MESSAGE;
}
