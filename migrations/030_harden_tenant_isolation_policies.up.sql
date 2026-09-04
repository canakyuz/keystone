-- 030: tenant izolasyon policy'lerini tek ve fail-closed bir forma getir.
--
-- Canli veritabanindan pg_policies dokumu alindi ve uc ayri hata sinifi bulundu.
--
-- 1) EKSIK CONTEXT'TE HATA (14 policy)
--    tenant_id = current_setting('app.current_tenant', TRUE)::UUID
--    Context sifirlandiginda current_setting bos string doner, '' ::UUID ise
--    "invalid input syntax for type uuid" firlatir. Veri sizmaz ama sorgu
--    anlasilmaz bir veritabani hatasiyla duser.
--
-- 2) AYARSIZ OTURUMDA SERT HATA (payments, payment_events, refunds)
--    current_setting('app.current_tenant') ikinci argumansiz cagriliyordu.
--    Parametre hic set edilmemisse "unrecognized configuration parameter"
--    hatasi verir.
--
-- 3) FAIL-OPEN (sites)  <-- en agiri
--    tenant_id = COALESCE(NULLIF(current_setting(...), '')::UUID, tenant_id)
--    Context yokken COALESCE tenant_id'ye duser ve kosul tenant_id = tenant_id
--    olur, yani her satir icin TRUE. Tenant context'i ayarlamayi unutan her
--    kod yolu sites tablosunun tamamini okuyabiliyordu.
--
-- ORTAK COZUM
--    tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID
--    Bos string NULL'a cevrilir, NULL karsilastirmasi hicbir satiri eslemez.
--    Context yoksa sonuc bos kumedir: hata yok, sizinti yok.
--
-- NOT: websites.public_websites_policy ve projects.public_projects_policy
-- bilincli olarak birakildi. Bunlar yayinlanmis icerigi tenant sinirindan
-- bagimsiz acan tasarim kararlaridir. Bkz. SECURITY.md.


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
