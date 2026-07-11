/**
 * Redirect-URI resolution for OAuthModal's "start login" step. Extracted to
 * keep OAuthModal within its frozen size budget and to make the per-provider
 * branching testable without rendering the component (it depends on
 * window.location in the component; here it's threaded in as plain params).
 */

export const GOOGLE_OAUTH_PROVIDERS = new Set(["antigravity", "agy", "gemini-cli"]);

/**
 * Providers whose upstream login endpoint has only ever been observed
 * accepting a loopback (127.0.0.1) callback_uri/redirect_uri — never confirmed
 * to accept a public HTTPS domain. Google's native/desktop OAuth client is
 * documented to be loopback-only. IBM Bob's own VS Code extension client
 * (app/extensions/bob-code) always hardcodes `http://127.0.0.1:<port>` when
 * building its login URL — it never tries a public domain, so bob.ibm.com's
 * server-side acceptance of anything else is unverified. Treat both the same
 * way: always request a loopback redirect_uri, even on a remote dashboard,
 * and let the user copy the (failed-to-load) callback URL from their
 * browser's address bar and paste it in manually — the same flow already
 * shipped for Google OAuth on remote installs.
 */
export const LOOPBACK_ONLY_OAUTH_PROVIDERS = new Set([...GOOGLE_OAUTH_PROVIDERS, "ibm-bob"]);

export interface RedirectUriContext {
  origin: string;
  port: string;
  protocol: string;
  isLocalhost: boolean;
  /** process.env.NEXT_PUBLIC_BASE_URL, inlined at build time in the browser bundle. */
  publicBaseUrl?: string | null;
}

/**
 * Resolve the redirect_uri to request for a given provider's OAuth flow.
 *
 * Strategy:
 * - Codex/OpenAI: fixed loopback port 1455 (registered in OAuth app).
 * - Windsurf/Devin CLI (remote fallback): loopback with OmniRoute's own port
 *   + /auth/callback (on true localhost the callback server handles it instead).
 * - LOOPBACK_ONLY_OAUTH_PROVIDERS (Google OAuth providers, IBM Bob): always
 *   loopback, even remotely — user copies the callback URL manually.
 * - Other providers, remote: actual origin (or NEXT_PUBLIC_BASE_URL override).
 * - Other providers, localhost: localhost:port.
 */
export function resolveOAuthModalRedirectUri(provider: string, ctx: RedirectUriContext): string {
  if (provider === "codex" || provider === "openai") {
    return "http://localhost:1455/auth/callback";
  }

  if (provider === "windsurf" || provider === "devin-cli") {
    const port = ctx.port || "20128";
    return `http://localhost:${port}/auth/callback`;
  }

  if (LOOPBACK_ONLY_OAUTH_PROVIDERS.has(provider)) {
    const port = ctx.port || "20128";
    return `http://127.0.0.1:${port}/callback`;
  }

  if (!ctx.isLocalhost) {
    const publicUrl = ctx.publicBaseUrl;
    const origin =
      publicUrl && publicUrl !== "http://localhost:20128"
        ? publicUrl.replace(/\/$/, "")
        : ctx.origin;
    return `${origin}/callback`;
  }

  const port = ctx.port || (ctx.protocol === "https:" ? "443" : "80");
  return `http://localhost:${port}/callback`;
}
