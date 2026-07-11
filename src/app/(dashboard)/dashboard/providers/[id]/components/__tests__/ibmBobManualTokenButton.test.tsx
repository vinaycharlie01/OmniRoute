// @vitest-environment jsdom
//
// ibm-bob's OAuth token exchange can fail against IBM's real backend for
// reasons outside OmniRoute's control. Both the toolbar and the empty-state
// placeholder must offer a secondary "Paste API Token" action (reusing the
// existing manual API-key modal via openApiKeyAddFlow) so a user with an
// already-working Bob Bearer token is never blocked on OAuth succeeding.
import React, { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, it, vi } from "vitest";
import ConnectionsHeaderToolbar from "../ConnectionsHeaderToolbar";
import EmptyConnectionsPlaceholder from "../EmptyConnectionsPlaceholder";

const cleanups: Array<() => void> = [];

function renderComponent(node: React.ReactElement) {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => root.render(node));
  cleanups.push(() => {
    act(() => root.unmount());
    container.remove();
  });
  return container;
}

afterEach(() => {
  while (cleanups.length) cleanups.pop()?.();
  document.body.innerHTML = "";
});

const t = Object.assign((k: string) => k, { has: () => false });

const baseToolbarProps = {
  providerInfo: { name: "IBM Bob" },
  isCommandCode: false,
  providerSupportsPat: false,
  connections: [],
  batchTesting: false,
  batchRetesting: false,
  retestingId: null,
  proxyConfig: {},
  preferClaudeCodeForUnprefixedClaudeModels: false,
  claudeRoutingSettingsLoaded: true,
  claudeRoutingSettingsLoadError: null,
  savingClaudeRoutingPreference: false,
  handleToggleClaudeRoutingPreference: vi.fn(),
  loadClaudeRoutingSettings: vi.fn(),
  codexGlobalServiceMode: "default",
  codexGlobalServiceModeOptions: [],
  codexSettingsLoaded: true,
  codexSettingsLoadError: null,
  savingCodexGlobalServiceMode: false,
  handleChangeCodexGlobalServiceMode: vi.fn(),
  loadCodexSettings: vi.fn(),
  onSetProxyTarget: vi.fn(),
  handleDistributeProxies: vi.fn(),
  handleBatchTestAll: vi.fn(),
  gateConnectionFlow: (cb: () => void) => cb(),
  openApiKeyAddFlow: vi.fn(),
  openPrimaryAddFlow: vi.fn(),
  openExternalLinkFlow: vi.fn(),
  handleOpenCommandCodeConnect: vi.fn(),
  commandCodeAuthState: { phase: "idle" },
  onOpenOAuthModal: vi.fn(),
  onOpenCodexCliGuide: vi.fn(),
  onOpenImportCodex: vi.fn(),
  onOpenImportClaude: vi.fn(),
  onOpenImportGemini: vi.fn(),
  onOpenImportGrokCli: vi.fn(),
  t,
};

const baseEmptyPlaceholderProps = {
  isCompatible: false,
  isCommandCode: false,
  providerSupportsPat: false,
  commandCodeAuthState: { phase: "idle" },
  gateConnectionFlow: (cb: () => void) => cb(),
  openApiKeyAddFlow: vi.fn(),
  openPrimaryAddFlow: vi.fn(),
  handleOpenCommandCodeConnect: vi.fn(),
  onOpenOAuthModal: vi.fn(),
  onOpenImportCodex: vi.fn(),
  onOpenImportClaude: vi.fn(),
  onOpenImportGemini: vi.fn(),
  onOpenImportGrokCli: vi.fn(),
  t,
};

describe("ibm-bob manual API token fallback button", () => {
  it("ConnectionsHeaderToolbar shows 'Paste API Token' for ibm-bob", () => {
    const openApiKeyAddFlow = vi.fn();
    const c = renderComponent(
      <ConnectionsHeaderToolbar
        {...baseToolbarProps}
        providerId="ibm-bob"
        isCompatible={false}
        isOAuth={true}
        openApiKeyAddFlow={openApiKeyAddFlow}
      />
    );
    const button = Array.from(c.querySelectorAll("button")).find((b) =>
      b.textContent?.includes("Paste API Token")
    );
    expect(button).toBeDefined();

    act(() => button?.dispatchEvent(new MouseEvent("click", { bubbles: true })));
    expect(openApiKeyAddFlow).toHaveBeenCalled();
  });

  it("ConnectionsHeaderToolbar does not show 'Paste API Token' for other OAuth providers", () => {
    const c = renderComponent(
      <ConnectionsHeaderToolbar
        {...baseToolbarProps}
        providerId="qoder"
        isCompatible={false}
        isOAuth={true}
      />
    );
    expect(c.textContent).not.toContain("Paste API Token");
    expect(c.textContent).toContain("Experimental OAuth");
  });

  it("EmptyConnectionsPlaceholder shows 'Paste API Token' for ibm-bob", () => {
    const openApiKeyAddFlow = vi.fn();
    const c = renderComponent(
      <EmptyConnectionsPlaceholder
        {...baseEmptyPlaceholderProps}
        providerId="ibm-bob"
        isOAuth={true}
        openApiKeyAddFlow={openApiKeyAddFlow}
      />
    );
    const button = Array.from(c.querySelectorAll("button")).find((b) =>
      b.textContent?.includes("Paste API Token")
    );
    expect(button).toBeDefined();

    act(() => button?.dispatchEvent(new MouseEvent("click", { bubbles: true })));
    expect(openApiKeyAddFlow).toHaveBeenCalled();
  });

  it("EmptyConnectionsPlaceholder does not show 'Paste API Token' for other OAuth providers", () => {
    const c = renderComponent(
      <EmptyConnectionsPlaceholder
        {...baseEmptyPlaceholderProps}
        providerId="qoder"
        isOAuth={true}
      />
    );
    expect(c.textContent).not.toContain("Paste API Token");
    expect(c.textContent).toContain("Experimental OAuth");
  });
});
