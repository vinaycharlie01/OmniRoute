import { decodeJwt } from "jose";
import { IBM_BOB_CONFIG } from "../constants/oauth";

/**
 * IBM Bob OAuth provider — authorization_code flow, no client_id/secret.
 *
 * Matches the newer Bob VS Code extension bundle's own client: it builds
 * `${webLoginUrl}/login?callback_uri=<redirect>&state=<uuid>`, then exchanges
 * the returned `code` with a bare `{code}` POST body (no PKCE code_verifier,
 * no client secret) against `${gatewayBaseUrl}/v1/auth/token`.
 */
export const ibmBob = {
  config: IBM_BOB_CONFIG,
  flowType: "authorization_code",
  buildAuthUrl: (config: typeof IBM_BOB_CONFIG, redirectUri: string, state: string) => {
    const params = new URLSearchParams({ callback_uri: redirectUri, state });
    return `${config.webLoginUrl}/login?${params.toString()}`;
  },
  exchangeToken: async (config: typeof IBM_BOB_CONFIG, code: string) => {
    const response = await fetch(config.tokenUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        "User-Agent": config.userAgent,
      },
      body: JSON.stringify({ code }),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`IBM Bob token exchange failed: ${error}`);
    }

    return await response.json();
  },
  mapTokens: (tokens: {
    token?: string;
    refresh_token?: string;
    idp_access_token?: string;
    idp_id_token?: string;
  }) => {
    let email: string | null = null;
    let displayName: string | null = null;
    let expiresIn: number | undefined;

    if (tokens.token) {
      try {
        const decoded = decodeJwt(tokens.token);
        email = (decoded.email as string) || (decoded.user as string) || null;
        displayName = (decoded.name as string) || email;
        if (typeof decoded.exp === "number") {
          expiresIn = Math.max(1, decoded.exp - Math.floor(Date.now() / 1000));
        }
      } catch {
        // Best-effort metadata only — missing/invalid claims don't block login.
      }
    }

    return {
      accessToken: tokens.token,
      refreshToken: tokens.refresh_token || null,
      expiresIn,
      email,
      displayName,
      providerSpecificData: {
        idpAccessToken: tokens.idp_access_token || null,
        idpIdToken: tokens.idp_id_token || null,
      },
    };
  },
};
