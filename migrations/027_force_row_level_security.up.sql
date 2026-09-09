-- 027: force Row Level Security on the table owner too.
--
-- SORUN
-- Every earlier migration used "ALTER TABLE x ENABLE ROW LEVEL SECURITY". In
-- PostgreSQL, ENABLE exempts the table's OWNER from the policies. Because the
-- application connects as the role that created the tables in nearly every setup
-- (including the docker-compose in this repository), the tenant isolation policies
-- were silently inactive.
--
-- MEASURED (as a non-superuser role that owns the tables)
--   ENABLE  -> both rows, from two different tenants, are visible
--   FORCE   -> only one is visible, so the policy is being applied
--
-- FIX
-- FORCE ROW LEVEL SECURITY subjects the owner to the policies as well.
--
-- REMAINING OPERATIONAL REQUIREMENT
-- Superusers bypass RLS under all circumstances and FORCE does not change that. The
-- application connection must NEVER be a superuser. See SECURITY.md.
--
-- NOTE: the tenants table is deliberately out of scope. It is control-plane data and
-- cannot be narrowed by tenant; its isolation happens in the application layer.

ALTER TABLE appointments     FORCE ROW LEVEL SECURITY;
ALTER TABLE assignments      FORCE ROW LEVEL SECURITY;
ALTER TABLE availabilities   FORCE ROW LEVEL SECURITY;
ALTER TABLE blog_categories  FORCE ROW LEVEL SECURITY;
ALTER TABLE blog_posts       FORCE ROW LEVEL SECURITY;
ALTER TABLE lessons          FORCE ROW LEVEL SECURITY;
ALTER TABLE payment_events   FORCE ROW LEVEL SECURITY;
ALTER TABLE payments         FORCE ROW LEVEL SECURITY;
ALTER TABLE projects         FORCE ROW LEVEL SECURITY;
ALTER TABLE refunds          FORCE ROW LEVEL SECURITY;
ALTER TABLE services         FORCE ROW LEVEL SECURITY;
ALTER TABLE sites            FORCE ROW LEVEL SECURITY;
ALTER TABLE students         FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_modules   FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_tools     FORCE ROW LEVEL SECURITY;
ALTER TABLE users            FORCE ROW LEVEL SECURITY;
ALTER TABLE websites         FORCE ROW LEVEL SECURITY;
