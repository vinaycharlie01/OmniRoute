// Regression guard for the Docker build's better-sqlite3 native-binding step
// (Dockerfile builder stage: `npm ci --ignore-scripts && npm rebuild
// better-sqlite3 && node -e "require('better-sqlite3')(...)"`).
//
// npm's script-allowlist gate (introduced npm 12) silently no-ops
// `npm rebuild better-sqlite3` unless package.json's top-level "allowScripts"
// has an entry pinned to the exact resolved version — without it, the native
// binding never gets built and the Dockerfile's smoke-test require() fails
// with "Could not locate the bindings file". This test fails the moment
// better-sqlite3 is bumped in package-lock.json without updating the pinned
// allowScripts entry to match, catching the drift before it reaches CI.
import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "..", "..");

test("package.json allowScripts pins the exact better-sqlite3 version from package-lock.json", () => {
  const pkg = JSON.parse(fs.readFileSync(path.join(ROOT, "package.json"), "utf8"));
  const lock = JSON.parse(fs.readFileSync(path.join(ROOT, "package-lock.json"), "utf8"));

  const lockedVersion = lock.packages?.["node_modules/better-sqlite3"]?.version;
  assert.ok(
    typeof lockedVersion === "string" && lockedVersion.length > 0,
    "package-lock.json must have a resolved node_modules/better-sqlite3 version"
  );

  const expectedKey = `better-sqlite3@${lockedVersion}`;
  assert.equal(
    pkg.allowScripts?.[expectedKey],
    true,
    `package.json "allowScripts" must contain "${expectedKey}": true — run ` +
      `\`npm install-scripts approve better-sqlite3\` after bumping the dependency`
  );
});
