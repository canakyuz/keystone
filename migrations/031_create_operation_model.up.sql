-- 031: Uzun suren islemler icin kalici operasyon modeli.
--
-- PROBLEM
-- Tenant kurulumu senkron calisiyordu: HTTP istegi semayi olusturana kadar
-- bekliyordu. Bunun iki sonucu vardi. Istek zaman asimina ugradiginda
-- istemcinin isin ne durumda oldugunu ogrenme yolu yoktu. Ve surec kurulum
-- ortasinda kapandiginda, yarim kalmis isi kimse devralamiyordu.
--
-- COZUM
-- Uc tablo: ne yapildigi (operations), kimin yapacagi (provisioning_jobs),
-- ve ayni istegin tekrar gelmesi durumunda ne olacagi (idempotency_keys).

-- 1) OPERATIONS -----------------------------------------------------------
--
-- Kullaniciya donuk kayittir. Istemci bu kaydin adresini alir ve durumu
-- buradan sorgular.
--
-- NEDEN tenant durumundan ayri: bir tenant'in durumu ile tek bir islemin
-- durumu ayni sey degildir. Basarisiz bir kurulumdan sonra ayni tenant icin
-- yeni bir operasyon acilabilir; tenant 'failed' kalirken yeni operasyon
-- 'pending' baslar. Ikisini tek kolonda tutmak bu ayrimi imkansiz kilardi.
CREATE TABLE IF NOT EXISTS operations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Islem turu, orn. 'tenant.provision'.
    kind VARCHAR(64) NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    -- Hata, makine tarafindan okunabilir kod ve insana donuk mesaj olarak
    -- ayri tutulur. Istemci koda gore dallanir, mesaji kullaniciya gosterir.
    error_code VARCHAR(64),
    error_message TEXT,

    created_by UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT operations_status_check
        CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),

    -- Tamamlanmis bir operasyonun bitis zamani olmak ZORUNDA.
    -- Bu kisit olmadan, kismi bir guncelleme sessizce tutarsiz kayit birakir.
    CONSTRAINT operations_completed_at_present
        CHECK (
            (status IN ('succeeded', 'failed') AND completed_at IS NOT NULL)
            OR (status IN ('pending', 'running') AND completed_at IS NULL)
        )
);

CREATE INDEX idx_operations_tenant ON operations(tenant_id, created_at DESC);
CREATE INDEX idx_operations_pending ON operations(status) WHERE status IN ('pending', 'running');

-- 2) PROVISIONING_JOBS ----------------------------------------------------
--
-- Worker'in devraldigi is birimidir. Operations kullaniciya, bu tablo
-- calistirana bakar.
CREATE TABLE IF NOT EXISTS provisioning_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operation_id UUID NOT NULL REFERENCES operations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,

    -- Isin ne zaman calistirilabilecegi. Yeniden denemede geriye alinir.
    next_attempt_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- LEASE
    -- Worker isi sonsuza kadar degil, sinirli sure icin sahiplenir.
    -- Kalici bir 'processing' bayragi kullanilsaydi, surec kurulum ortasinda
    -- kapandiginda is sonsuza kadar o bayrakla kalirdi ve kimse devralamazdi.
    -- Lease suresi dolan is yeniden sahiplenilebilir.
    lease_owner VARCHAR(128),
    lease_expires_at TIMESTAMP WITH TIME ZONE,

    -- FENCING TOKEN
    -- Her sahiplenmede artan sayac.
    --
    -- NEDEN gerekli: lease suresi dolup is baska bir worker'a gectikten
    -- sonra, eski worker geri donup "tamamlandi" bildirebilir. Yalnizca
    -- lease_owner'a bakmak yetmez; eski worker kendi adini bilir. Sonuc
    -- bildirimi guncel fence degeriyle yapilmak zorundadir, boylece
    -- gecikmis bildirim reddedilir.
    fence BIGINT NOT NULL DEFAULT 0,

    last_error TEXT,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT provisioning_jobs_status_check
        CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'dead')),

    CONSTRAINT provisioning_jobs_attempts_bounded
        CHECK (attempts >= 0 AND attempts <= max_attempts),

    -- Calisan bir isin sahibi ve lease bitisi olmak zorunda.
    CONSTRAINT provisioning_jobs_running_has_lease
        CHECK (
            status <> 'running'
            OR (lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
        )
);

-- Claim sorgusunun taradigi kume: yalnizca calistirilabilir isler.
-- Partial index, tamamlanmis isleri tarama disinda birakir; tablo buyudukce
-- claim maliyeti sabit kalir.
CREATE INDEX idx_jobs_claimable
    ON provisioning_jobs(next_attempt_at)
    WHERE status IN ('pending', 'running');

CREATE INDEX idx_jobs_operation ON provisioning_jobs(operation_id);

-- Bir operasyonun ayni anda birden fazla aktif isi olamaz.
CREATE UNIQUE INDEX idx_jobs_one_active_per_operation
    ON provisioning_jobs(operation_id)
    WHERE status IN ('pending', 'running');

-- 3) IDEMPOTENCY_KEYS -----------------------------------------------------
--
-- Ayni istegin tekrar gelmesi durumunda ayni mantiksal islemi dondurur.
--
-- NEDEN unique constraint: "once var mi diye bak, yoksa ekle" dizisi tek
-- basina yaris kosulunu engellemez. Iki istek ayni anda kontrolu gecip iki
-- ayri operasyon yaratabilir. Benzersizligi veritabani zorlar; uygulama
-- ihlali yakalayip mevcut kaydi dondurur.
CREATE TABLE IF NOT EXISTS idempotency_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Anahtarin gecerli oldugu kapsam. Bir musterinin anahtari digerinin
    -- istegini eslestirmemelidir.
    scope VARCHAR(128) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,

    -- Istegin normalize edilmis ozeti.
    -- Ayni anahtar farkli govdeyle gelirse cakisma bildirilir; sessizce
    -- eski sonucu dondurmek istemciyi yanlis yonlendirirdi.
    request_fingerprint CHAR(64) NOT NULL,

    operation_id UUID NOT NULL REFERENCES operations(id) ON DELETE CASCADE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Saklama suresi. Bu sure dolduktan sonra ayni anahtar yeni bir islem
    -- yaratir; yani idempotency garantisi suresizdir degildir ve bu sure
    -- API dokumantasyonunda belirtilmelidir.
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE UNIQUE INDEX idx_idempotency_scope_key
    ON idempotency_keys(scope, idempotency_key);

CREATE INDEX idx_idempotency_expiry ON idempotency_keys(expires_at);

-- RLS: operasyon ve is kayitlari kiraci verisidir.
-- Migration 030'daki fail-closed forma uyar.
ALTER TABLE operations ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON operations
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE provisioning_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE provisioning_jobs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON provisioning_jobs
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

COMMENT ON COLUMN provisioning_jobs.fence IS
    'Her sahiplenmede artar. Gecikmis worker bildirimini reddetmek icin kullanilir.';
COMMENT ON COLUMN provisioning_jobs.lease_expires_at IS
    'Lease bitisi. Suresi dolan is baska bir worker tarafindan devralinabilir.';
