-- 030 geri alma: policy'ler onceki (hatali) formlarina donmez;
-- guvenli forma esdeger sade forma indirilir.


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
