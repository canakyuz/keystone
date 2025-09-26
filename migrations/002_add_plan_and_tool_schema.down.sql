-- Migration: 002_add_plan_and_tool_schema (down)
-- Description: Reverts the changes made in the up migration.

BEGIN;

DROP TABLE IF EXISTS translations;
DROP TABLE IF EXISTS languages;
DROP TABLE IF EXISTS sites;
DROP TABLE IF EXISTS tenant_tool_settings;
DROP TABLE IF EXISTS plan_tools;
DROP TABLE IF EXISTS tools;
DROP TABLE IF EXISTS plans;

COMMIT;
