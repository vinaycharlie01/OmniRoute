import test from "node:test";
import assert from "node:assert/strict";

import { REGISTRY } from "../../open-sse/config/providerRegistry.ts";
import { getModelsByProviderId } from "../../src/shared/constants/models.ts";
import { APIKEY_PROVIDERS, AI_PROVIDERS } from "../../src/shared/constants/providers.ts";

const CHAT_OPENAI_COMPAT_PROVIDER_IDS = [
  "deepinfra",
  "vercel-ai-gateway",
  "qianfan",
  "lambda-ai",
  "sambanova",
  "nscale",
  "ovhcloud",
  "baseten",
  "publicai",
  "moonshot",
  "meta-llama",
  "v0-vercel",
  "morph",
  "featherless-ai",
  "friendliai",
  "llamagate",
  "heroku",
  "galadriel",
  "databricks",
  "snowflake",
  "wandb",
  "volcengine",
  "ai21",
  "gigachat",
  "venice",
  "codestral",
  "upstage",
  "maritalk",
  "xiaomi-mimo",
  "inference-net",
  "nanogpt",
  "predibase",
  "bytez",
  "reka",
  "byteplus",
  "orcarouter",
];

// ibm-bob is checked separately below: it's OAuth-primary (recategorized so
// bob.ibm.com/login is the default sign-in path), so its UI metadata lives in
// AI_PROVIDERS (backed by OAUTH_PROVIDERS), not APIKEY_PROVIDERS like the
// providers above — but the chat request shape (registry/models) is identical.

test("chat-openai-compat providers are registered across provider metadata, registry and local catalog", () => {
  for (const providerId of CHAT_OPENAI_COMPAT_PROVIDER_IDS) {
    assert.ok(APIKEY_PROVIDERS[providerId], `${providerId} missing from APIKEY_PROVIDERS`);
    assert.ok(REGISTRY[providerId], `${providerId} missing from REGISTRY`);

    const models = getModelsByProviderId(providerId);
    assert.ok(Array.isArray(models), `${providerId} models must be an array`);
    assert.ok(models.length > 0, `${providerId} models must not be empty`);
  }
});

test("orcarouter models keep the orcarouter/ namespace prefix and enable passthrough", () => {
  const modelIds = REGISTRY.orcarouter.models.map((model) => model.id);

  // OrcaRouter's distributor matches channels by the namespaced id, so a bare
  // "auto" returns 503 "No available channel" — the router id must stay prefixed.
  assert.ok(modelIds.includes("orcarouter/auto"), "expected namespaced orcarouter/auto");
  assert.equal(modelIds.includes("auto"), false, "bare 'auto' would 503 upstream");

  // Pinned vendor models also carry their upstream namespace.
  assert.ok(modelIds.includes("anthropic/claude-opus-4.8"));

  // The 150+ catalog beyond the curated flagship list is reachable via passthrough.
  assert.equal(APIKEY_PROVIDERS.orcarouter.passthroughModels, true);
});

test("ibm-bob is a Bearer-auth OpenAI-compatible gateway with passthrough enabled", () => {
  assert.equal(REGISTRY["ibm-bob"].format, "openai");
  assert.equal(REGISTRY["ibm-bob"].authType, "apikey");
  assert.equal(REGISTRY["ibm-bob"].authHeader, "bearer");
  assert.equal(REGISTRY["ibm-bob"].baseUrl, "https://api.us-east.bob.ibm.com/v1/chat/completions");

  const modelIds = REGISTRY["ibm-bob"].models.map((model) => model.id);
  assert.ok(modelIds.includes("premium"), "expected the default 'premium' model alias");
  assert.equal(AI_PROVIDERS["ibm-bob"].passthroughModels, true);

  const models = getModelsByProviderId("ibm-bob");
  assert.ok(Array.isArray(models) && models.length > 0, "ibm-bob models must not be empty");
});

test("upstage chat catalog does not include non-chat specialty models", () => {
  const modelIds = REGISTRY.upstage.models.map((model) => model.id);

  assert.ok(modelIds.includes("solar-pro3"));
  assert.ok(modelIds.includes("solar-mini"));
  assert.equal(modelIds.includes("document-parse"), false);
  assert.equal(modelIds.includes("embedding-query"), false);
  assert.equal(modelIds.includes("embedding-passage"), false);
});
