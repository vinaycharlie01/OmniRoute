/**
 * bob (formerly registered as "ibm-bob") had no entry in
 * PROVIDER_MODELS_CONFIG and doesn't match any of the isOpenAICompatibleProvider
 * / isLocalOpenAIStyleProvider / isNamedOpenAIStyleProvider live-fetch branches,
 * so GET /api/providers/[id]/models always fell straight to the registry's
 * small hardcoded catalog (reported as "API unavailable — using local catalog",
 * even though no fetch was ever attempted).
 *
 * A live `GET /inference/v1/model/info` call (LiteLLM-style, confirmed working
 * with a real key: `{ data: [{ model_name, model_info }] }`) is Bob's real
 * catalog endpoint — traced from the Bob VS Code extension's own
 * ModelInfoService (`this.gatewayClient.fetch("/model/info")` on the inference
 * gateway client). PROVIDER_MODELS_CONFIG.bob now wires this up so the model
 * list stays fresh instead of only ever showing the seeded catalog.
 */
import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

const TEST_DATA_DIR = fs.mkdtempSync(path.join(os.tmpdir(), "omniroute-bob-model-info-"));
process.env.DATA_DIR = TEST_DATA_DIR;

const core = await import("../../src/lib/db/core.ts");
const providersDb = await import("../../src/lib/db/providers.ts");
const modelsRoute = await import("../../src/app/api/providers/[id]/models/route.ts");

async function resetStorage() {
  core.resetDbInstance();
  fs.rmSync(TEST_DATA_DIR, { recursive: true, force: true });
  fs.mkdirSync(TEST_DATA_DIR, { recursive: true });
}

test.after(() => {
  core.resetDbInstance();
  fs.rmSync(TEST_DATA_DIR, { recursive: true, force: true });
});

interface ModelsBody {
  provider: string;
  connectionId: string;
  models: Array<{ id: string }>;
  source?: string;
}

test("bob fetches the live /inference/v1/model/info catalog with an x-api-key header", async () => {
  await resetStorage();
  const connection = await providersDb.createProviderConnection({
    provider: "bob",
    authType: "apikey",
    name: "bob-live",
    apiKey: "bob_prod_test-key",
  });

  let requestedUrl = "";
  let requestedApiKeyHeader: string | null = null;
  const originalFetch = globalThis.fetch;
  globalThis.fetch = async (url, init) => {
    requestedUrl = String(url);
    requestedApiKeyHeader = new Headers(init?.headers).get("x-api-key");
    return Response.json({
      data: [
        { model_name: "premium", model_info: {} },
        { model_name: "sonnet-4.6", model_info: {} },
        { model_name: "gpt-oss-20b", model_info: {} },
      ],
    });
  };

  try {
    const response = await modelsRoute.GET(
      new Request(`http://localhost/api/providers/${connection.id}/models?refresh=true`),
      { params: { id: connection.id } }
    );
    assert.equal(response.status, 200);
    const body = (await response.json()) as ModelsBody;
    assert.equal(body.provider, "bob");
    assert.equal(body.source, "api", "should serve the live upstream catalog, not local_catalog");
    assert.equal(requestedUrl, "https://api.us-east.bob.ibm.com/inference/v1/model/info");
    assert.equal(requestedApiKeyHeader, "bob_prod_test-key");
    const ids = body.models.map((m) => m.id);
    assert.ok(ids.includes("sonnet-4.6"), `live ids missing: ${ids.join(",")}`);
    assert.ok(ids.includes("gpt-oss-20b"), `live ids missing: ${ids.join(",")}`);
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test("bob falls back to the local catalog when the live model/info fetch fails", async () => {
  await resetStorage();
  const connection = await providersDb.createProviderConnection({
    provider: "bob",
    authType: "apikey",
    name: "bob-fallback",
    apiKey: "bob_prod_test-key-2",
  });

  const originalFetch = globalThis.fetch;
  globalThis.fetch = async () => new Response("bad gateway", { status: 502 });

  try {
    const response = await modelsRoute.GET(
      new Request(`http://localhost/api/providers/${connection.id}/models?refresh=true`),
      { params: { id: connection.id } }
    );
    assert.equal(response.status, 200);
    const body = (await response.json()) as ModelsBody;
    assert.equal(body.provider, "bob");
    assert.equal(body.source, "local_catalog", "import must not break when upstream is down");
    assert.ok(body.models.length > 0, "fallback catalog should be non-empty");
    assert.ok(body.models.some((m) => m.id === "premium"));
  } finally {
    globalThis.fetch = originalFetch;
  }
});
