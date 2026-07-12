-- Migration: rename provider id "ibm-bob" -> "bob"
--
-- ibm-bob's real request path (/inference/v1, x-api-key) and API-key-primary
-- categorization were confirmed live; the provider id itself is renamed to
-- "bob" to match the shorter, less IBM-specific branding used going forward.
-- Only LIVE/operational state is migrated here so an existing connection keeps
-- working under the new id. Historical/analytics tables (request logs, call
-- log summaries, compression analytics, session model history, quota
-- snapshots) intentionally keep the old "ibm-bob" id as an accurate record of
-- what was true at the time — they are not rewritten.

UPDATE provider_connections SET provider = 'bob' WHERE provider = 'ibm-bob';
UPDATE domain_circuit_breakers SET name = 'bob' WHERE name = 'ibm-bob';
UPDATE session_account_affinity SET provider = 'bob' WHERE provider = 'ibm-bob';
UPDATE tier_assignments SET provider = 'bob' WHERE provider = 'ibm-bob';
UPDATE group_model_permissions SET provider = 'bob' WHERE provider = 'ibm-bob';
UPDATE provider_plans SET provider = 'bob' WHERE provider = 'ibm-bob';
UPDATE provider_quota_reset_events SET provider = 'bob' WHERE provider = 'ibm-bob';
UPDATE provider_key_limits SET provider = 'bob' WHERE provider = 'ibm-bob';
