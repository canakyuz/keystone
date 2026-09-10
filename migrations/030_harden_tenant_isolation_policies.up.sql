-- 030: bring the tenant isolation policies into one fail-closed form.
--
-- A pg_policies dump was taken from the live database and three classes of bug were
-- found.
--
-- 1) ERROR WHEN THE CONTEXT IS MISSING (14 policies)
--    tenant_id = current_setting('app.current_tenant', TRUE)::UUID
--    With the context cleared, current_setting returns an empty string, and ''::UUID
--    raises "invalid input syntax for type uuid". No data leaks, but the query fails
--    with an incomprehensible database error.
--
-- 2) HARD ERROR ON AN UNCONFIGURED SESSION (payments, payment_events, refunds)
--    current_setting('app.current_tenant') was called without its second argument.
--    If the parameter was never set, it raises "unrecognized configuration
--    parameter".
--
-- 3) FAIL-OPEN (sites)  <-- the worst of them
--    tenant_id = COALESCE(NULLIF(current_setting(...), '')::UUID, tenant_id)
--    With no context, COALESCE falls back to tenant_id and the condition becomes
--    tenant_id = tenant_id, that is, TRUE for every row. Any code path that forgot to
--    set the tenant context could read the whole sites table.
--
-- THE COMMON FIX
--    tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID
--    The empty string becomes NULL, and a NULL comparison matches no row. With no
--    context the result is the empty set: no error, no leak.
--
-- NOTE: websites.public_websites_policy and projects.public_projects_policy were left
-- alone deliberately. They are design decisions that open published content across the
-- tenant boundary. See SECURITY.md.


DROP POLICY IF EXISTS tenant_isolation_policy ON appointments;
CREATE POLICY tenant_isolation_policy ON appointments
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_policy ON assignments;
CREATE POLICY tenant_isolation_policy ON assignments
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_policy ON availabilities;
CREATE POLICY tenant_isolation_policy ON availabilities
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_policy ON blog_categories;
CREATE POLICY tenant_isolation_policy ON blog_categories
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_policy ON blog_posts;
CREATE POLICY tenant_isolation_policy ON blog_posts
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_policy ON lessons;
CREATE POLICY tenant_isolation_policy ON lessons
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_payment_events ON payment_events;
CREATE POLICY tenant_isolation_payment_events ON payment_events
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_payments ON payments;
CREATE POLICY tenant_isolation_payments ON payments
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_policy ON projects;
CREATE POLICY tenant_isolation_policy ON projects
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_refunds ON refunds;
CREATE POLICY tenant_isolation_refunds ON refunds
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_policy ON services;
CREATE POLICY tenant_isolation_policy ON services
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_policy ON sites;
CREATE POLICY tenant_isolation_policy ON sites
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_policy ON students;
CREATE POLICY tenant_isolation_policy ON students
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_modules ON tenant_modules;
CREATE POLICY tenant_isolation_modules ON tenant_modules
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_tools ON tenant_tools;
CREATE POLICY tenant_isolation_tools ON tenant_tools
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_policy ON users;
CREATE POLICY tenant_isolation_policy ON users
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

DROP POLICY IF EXISTS tenant_isolation_policy ON websites;
CREATE POLICY tenant_isolation_policy ON websites
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);
