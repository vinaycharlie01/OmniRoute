# ── Common base with runtime deps (Alpine/musl — lean default path) ────────
#
# Alpine cuts the base OS layer dramatically vs. Debian-slim (musl libc,
# busybox, apk instead of a full Debian userland). better-sqlite3 is a native
# addon and MUST be compiled for the libc it will run under — the
# npm rebuild step below already forces a from-source compile (never a
# downloaded prebuilt), so it produces a musl-correct binary here automatically.
#
# runner-web (Playwright/Chromium) does NOT extend this chain: Playwright's
# Chromium builds are not supported on Alpine/musl (upstream limitation), so
# it has its own glibc (Debian-slim) chain below — see "base-glibc". Keep the
# two chains independent; don't try to make runner-web depend on this stage.
FROM node:24-alpine AS base
WORKDIR /app

# `apk upgrade` pulls security-patched versions of the Alpine base-image
# packages at build time, mirroring the Debian `apt-get upgrade` hygiene this
# image used to rely on. `--no-cache` fetches a temporary index and discards
# it in the same layer, so there's no separate lists-cleanup step needed
# (unlike apt).
RUN apk update \
  && apk upgrade --no-cache \
  && apk add --no-cache libsecret ca-certificates

# Refresh the globally-installed npm so its *bundled* node_modules (undici, tar)
# ship the patched versions. These are npm's own internals — not application
# dependencies (our app already resolves undici@8.5.0 / tar@7.5.16, both fixed) —
# but the container scanner flags the stale copies under
# /usr/local/lib/node_modules/npm/node_modules. npm is not invoked at runtime in
# the runner stages, so this is hygiene, not an exploitable runtime path.
RUN npm install -g npm@latest \
  && npm cache clean --force

# ── Builder (Alpine) ────────────────────────────────────────────────────────
FROM base AS builder

# Build tools for native module compilation (node-gyp needs these on musl too).
RUN apk add --no-cache python3 make g++

COPY package*.json ./
# Workspace package manifests MUST be present before `npm ci` so npm materializes
# the workspace and installs its *workspace-only* deps (e.g. safe-regex,
# @toon-format/toon — declared in open-sse/package.json, not hoisted to root).
# Without this, `npm ci` skips them and `npm run build` fails with "Module not
# found" (root cause of the v3.8.39 Docker build break). workspaces = ["open-sse"].
COPY open-sse/package.json ./open-sse/package.json
COPY scripts/build/postinstall.mjs ./scripts/build/postinstall.mjs
COPY scripts/build/postinstallSupport.mjs ./scripts/build/postinstallSupport.mjs
COPY scripts/build/native-binary-compat.mjs ./scripts/build/native-binary-compat.mjs
ENV NPM_CONFIG_LEGACY_PEER_DEPS=true
# --ignore-scripts blocks broad dependency install/postinstall hooks, closing
# the supply-chain attack surface where a transitive dep can run arbitrary code
# at install time. better-sqlite3 still needs a native binding for the target
# platform, so rebuild and smoke-test only that known runtime dependency below.
# npm's script-allowlist gate (introduced npm 12) blocks `npm rebuild <pkg>`
# too, not just `ci` — better-sqlite3's node-gyp rebuild is a silent no-op
# without a matching entry in package.json's top-level "allowScripts", so the
# smoke-test node -e require() below fails with "Could not locate the
# bindings file" if that entry is missing or its pinned version is stale.
#
# We REQUIRE a committed package-lock.json so resolved dependency versions
# are reproducible.
RUN test -f package-lock.json \
  || (echo "package-lock.json is required for reproducible Docker builds" >&2 && exit 1)
RUN npm ci --no-audit --no-fund --legacy-peer-deps --ignore-scripts \
  && npm rebuild better-sqlite3 \
  && node -e "require('better-sqlite3')(':memory:').close()"

# Build with webpack (stable). Turbopack hit a non-recoverable internal panic on this
# Next.js version during the v3.8.27 release build — TurbopackInternalError "entered
# unreachable code: there must be a path to a root" in ImportTracer::get_traces, on both
# linux/amd64 and linux/arm64. Webpack is the proven engine (build:release / VPS / CI Build
# all green). Re-enable Turbopack (=1) once the upstream tracer bug is fixed.
# See docs/ops/QUALITY_GATE_PLAYBOOK.md Parte 6.
ENV OMNIROUTE_USE_TURBOPACK=0

# Raise the V8 heap ceiling for the build. The webpack production optimization
# pass (forced above since Turbopack panics) needs more than V8's default ceiling
# (~2 GB) for a codebase this size; a memory-constrained Docker build otherwise
# dies with "FATAL ERROR: ... JavaScript heap out of memory" at `[builder] npm run
# build` (#4076). NODE_OPTIONS propagates to the spawned `next build` child
# (build-next-isolated.mjs → resolveNextBuildEnv spreads process.env). Build-only;
# the runtime heap is set separately on the runner stage (OMNIROUTE_MEMORY_MB).
# Override for hosts with more/less RAM: `--build-arg OMNIROUTE_BUILD_MEMORY_MB=6144`.
ARG OMNIROUTE_BUILD_MEMORY_MB=4096
ENV NODE_OPTIONS="--max-old-space-size=${OMNIROUTE_BUILD_MEMORY_MB}"

COPY . ./
RUN mkdir -p /app/data && npm run build

# ── Runner base (Alpine) ─────────────────────────────────────────────────────
FROM base AS runner-base

LABEL org.opencontainers.image.title="omniroute" \
  org.opencontainers.image.description="Unified AI proxy — route any LLM through one endpoint" \
  org.opencontainers.image.url="https://omniroute.online" \
  org.opencontainers.image.source="https://github.com/diegosouzapw/OmniRoute" \
  org.opencontainers.image.licenses="MIT"

ENV NODE_ENV=production
ENV PORT=20128
ENV HOSTNAME=0.0.0.0
ENV HOST=0.0.0.0
ENV OMNIROUTE_MEMORY_MB=1024
ENV NODE_OPTIONS="--max-old-space-size=${OMNIROUTE_MEMORY_MB}"

# Data directory inside Docker — must match the volume mount in docker-compose.yml
ENV DATA_DIR=/app/data
RUN mkdir -p /app/data

# `npm run build` (build-next-isolated → assembleStandalone) bundles ALL runtime
# files into .build/next/standalone/ — .next, node_modules, migrations, scripts,
# docs, and the previously hand-COPY'd modules below (@swc/helpers, pino-*, split2,
# migrations). assembleStandalone copies them straight from the builder's
# node_modules, so they are present regardless of NFT/Turbopack trace behaviour.
# The old per-module overrides were therefore pure duplication and were removed
# (build-output-isolation cleanup). See scripts/build/assembleStandalone.mjs
# (EXTRA_MODULE_ENTRIES) for the single source of truth.
COPY --from=builder /app/.build/next/standalone ./
# better-sqlite3 is the one exception still copied explicitly: assembleStandalone
# only syncs its native build/ dir; the JS wrapper (lib/, package.json) is left to
# Next.js tracing. bootstrap-env requires SQLite BEFORE the standalone server
# starts, so guarantee the complete package independent of trace behaviour.
COPY --from=builder /app/node_modules/better-sqlite3 ./node_modules/better-sqlite3
# migrations land at <standalone>/migrations via assembleStandalone; point the runtime at them.
ENV OMNIROUTE_MIGRATIONS_DIR=/app/migrations

# Runtime scripts — assembleStandalone should copy these, but ensure they're present.
# If assembleStandalone ran correctly, these COPY commands are no-ops (files already exist).
# If it didn't run or partially failed, these serve as fallback copies to keep the image bootable.
COPY --from=builder /app/scripts/dev/run-standalone.mjs ./dev/run-standalone.mjs
COPY --from=builder /app/scripts/build/runtime-env.mjs ./build/runtime-env.mjs
COPY --from=builder /app/scripts/build/bootstrap-env.mjs ./build/bootstrap-env.mjs
COPY --from=builder /app/scripts/dev/standalone-server-ws.mjs ./server-ws.mjs
COPY --from=builder /app/scripts/dev/peer-stamp.mjs ./peer-stamp.mjs
COPY --from=builder /app/scripts/dev/http-method-guard.cjs ./http-method-guard.cjs
COPY --from=builder /app/scripts/dev/responses-ws-proxy.mjs ./responses-ws-proxy.mjs
COPY --from=builder /app/scripts/dev/webdav-handler.mjs ./webdav-handler.mjs

# Docker healthcheck script — not traced by Next.js standalone output, so copy
# it explicitly. The HEALTHCHECK CMD references it as `node healthcheck.mjs`.
COPY --from=builder /app/scripts/dev/healthcheck.mjs ./healthcheck.mjs

# Hand /app over to the baked-in `node` non-root user (UID/GID 1000 — the
# official node:alpine image ships the same node user/group as the Debian
# variant) so the runtime process never holds root privileges. The chown
# happens after all COPYs so it covers files originally owned by root in the
# builder stage.
RUN chown -R node:node /app

EXPOSE 20128

# Drop to non-root before ENTRYPOINT/CMD so every derived stage (runner-cli)
# also runs as a non-root user unless it explicitly switches back.
USER node

# Warns if the mounted data volume has wrong ownership. Already POSIX
# /bin/sh-only (no bashisms), so it runs unmodified under Alpine's busybox ash.
COPY --chmod=755 scripts/check-permissions.sh /tmp/check-permissions.sh
ENTRYPOINT ["/tmp/check-permissions.sh"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD ["node", "healthcheck.mjs"]

CMD ["node", "dev/run-standalone.mjs"]

FROM runner-base AS runner-cli

# Drop back to root briefly so we can install system + global npm packages,
# then return to the `node` non-root user before the CMD inherited from
# runner-base runs.
USER root

# Install system dependencies required by openclaw (git+ssh references).
# docker-cli / docker-cli-compose are the Alpine package names for the Docker
# CLI client and the `docker compose` v2 plugin (no daemon is run in-container).
RUN apk add --no-cache git ca-certificates docker-cli docker-cli-compose \
  && git config --system url."https://github.com/".insteadOf "ssh://git@github.com/"

# Install CLI tools globally. Separate layer from apt for better cache reuse.
# NOTE: these packages were previously only ever installed on glibc (Debian);
# if any ship a native postinstall step that assumes glibc, re-validate this
# stage specifically after the Alpine switch.
RUN npm install -g --no-audit --no-fund @openai/codex @anthropic-ai/claude-code droid openclaw@latest

USER node

# ── glibc chain (Debian-slim) — Playwright/Chromium is not supported on
# Alpine/musl, so runner-web gets its own independent build from here down.
# This duplicates the base+builder setup above; that duplication is the actual
# cost of keeping web-cookie providers working while the default images move
# to Alpine. Do not attempt to merge this back into the Alpine chain.
# ─────────────────────────────────────────────────────────────────────────
FROM node:24-trixie-slim AS base-glibc
WORKDIR /app

RUN apt-get update \
  && apt-get upgrade -y \
  && apt-get install -y --no-install-recommends libsecret-1-0 ca-certificates \
  && rm -rf /var/lib/apt/lists/*

RUN npm install -g npm@latest \
  && npm cache clean --force

FROM base-glibc AS builder-glibc

RUN apt-get update \
  && apt-get install -y --no-install-recommends python3 make g++ \
  && rm -rf /var/lib/apt/lists/*

COPY package*.json ./
COPY open-sse/package.json ./open-sse/package.json
COPY scripts/build/postinstall.mjs ./scripts/build/postinstall.mjs
COPY scripts/build/postinstallSupport.mjs ./scripts/build/postinstallSupport.mjs
COPY scripts/build/native-binary-compat.mjs ./scripts/build/native-binary-compat.mjs
ENV NPM_CONFIG_LEGACY_PEER_DEPS=true
RUN test -f package-lock.json \
  || (echo "package-lock.json is required for reproducible Docker builds" >&2 && exit 1)
RUN npm ci --no-audit --no-fund --legacy-peer-deps --ignore-scripts \
  && npm rebuild better-sqlite3 \
  && node -e "require('better-sqlite3')(':memory:').close()"

ENV OMNIROUTE_USE_TURBOPACK=0
ARG OMNIROUTE_BUILD_MEMORY_MB=4096
ENV NODE_OPTIONS="--max-old-space-size=${OMNIROUTE_BUILD_MEMORY_MB}"

COPY . ./
RUN mkdir -p /app/data && npm run build

# ── Runner Web (web-cookie providers: Gemini Web, Claude Turnstile) ───────────
#
#  Three image flavors:
#    runner-base  →  omniroute:VERSION        Lean Alpine base. No browsers.
#    runner-cli   →  omniroute:VERSION-cli     +codex/claude-code/droid/openclaw CLIs (Alpine).
#    runner-web   →  omniroute:VERSION-web     +Chromium/Playwright (glibc/Debian-slim).
#
#  Use runner-web when you need web-cookie providers (gemini-web, claude-web,
#  claude-turnstile). For all other providers runner-base is sufficient.
#
#  Build:
#    docker build --target runner-web -t omniroute:web .
#  Compose:
#    build:
#      context: .
#      target: runner-web
FROM base-glibc AS runner-web

LABEL org.opencontainers.image.title="omniroute" \
  org.opencontainers.image.description="Unified AI proxy — route any LLM through one endpoint" \
  org.opencontainers.image.url="https://omniroute.online" \
  org.opencontainers.image.source="https://github.com/diegosouzapw/OmniRoute" \
  org.opencontainers.image.licenses="MIT"

ENV NODE_ENV=production
ENV PORT=20128
ENV HOSTNAME=0.0.0.0
ENV HOST=0.0.0.0
ENV OMNIROUTE_MEMORY_MB=1024
ENV NODE_OPTIONS="--max-old-space-size=${OMNIROUTE_MEMORY_MB}"
ENV DATA_DIR=/app/data
RUN mkdir -p /app/data

COPY --from=builder-glibc /app/.build/next/standalone ./
COPY --from=builder-glibc /app/node_modules/better-sqlite3 ./node_modules/better-sqlite3
ENV OMNIROUTE_MIGRATIONS_DIR=/app/migrations

COPY --from=builder-glibc /app/scripts/dev/run-standalone.mjs ./dev/run-standalone.mjs
COPY --from=builder-glibc /app/scripts/build/runtime-env.mjs ./build/runtime-env.mjs
COPY --from=builder-glibc /app/scripts/build/bootstrap-env.mjs ./build/bootstrap-env.mjs
COPY --from=builder-glibc /app/scripts/dev/standalone-server-ws.mjs ./server-ws.mjs
COPY --from=builder-glibc /app/scripts/dev/peer-stamp.mjs ./peer-stamp.mjs
COPY --from=builder-glibc /app/scripts/dev/http-method-guard.cjs ./http-method-guard.cjs
COPY --from=builder-glibc /app/scripts/dev/responses-ws-proxy.mjs ./responses-ws-proxy.mjs
COPY --from=builder-glibc /app/scripts/dev/webdav-handler.mjs ./webdav-handler.mjs
COPY --from=builder-glibc /app/scripts/dev/healthcheck.mjs ./healthcheck.mjs

RUN chown -R node:node /app

EXPOSE 20128

# Copy playwright and playwright-core from the glibc builder stage.
# The slim runtime image does not have playwright in node_modules, so npx falls
# back to a registry download — unreliable on CI runners (exits 127 on failure).
# Copying from the builder avoids any network access at image-build time and also
# ensures the same playwright version is available at runtime for web-session providers.
COPY --from=builder-glibc /app/node_modules/playwright-core ./node_modules/playwright-core
COPY --from=builder-glibc /app/node_modules/playwright ./node_modules/playwright

# Install Playwright browser binaries + OS dependencies under root, then hand
# ownership of the browsers cache to the node user.
# PLAYWRIGHT_BROWSERS_PATH overrides the default ~/.cache/ms-playwright so the
# browsers land under /home/node which persists across image layers and is
# accessible to the non-root runtime user.
ENV PLAYWRIGHT_BROWSERS_PATH=/home/node/.cache/ms-playwright
RUN apt-get update \
  && node node_modules/playwright/cli.js install chromium --with-deps \
  && chown -R node:node /home/node/.cache \
  && rm -rf /var/lib/apt/lists/*

USER node

COPY --chmod=755 scripts/check-permissions.sh /tmp/check-permissions.sh
ENTRYPOINT ["/tmp/check-permissions.sh"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD ["node", "healthcheck.mjs"]

CMD ["node", "dev/run-standalone.mjs"]
