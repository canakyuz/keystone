-- 032: Tenant yasam dongusune kurulum durumlarini ekle.
--
-- PROBLEM
-- Sema yalnizca dort duruma izin veriyordu: active, suspended, inactive, trial.
-- Tenant olusturma senkron oldugu surece bu yetiyordu; tenant ya vardi ya yoktu.
--
-- Kurulum asenkron hale gelince arada gecen sure gorunur oldu. Bir tenant
-- kaydi olusmus ama semasi henuz hazir degilse, durumu 'active' olamaz:
-- istekleri kabul edilirse var olmayan bir semaya yonlendirilir.
--
-- DURUM MAKINESI
--
--   pending ---> provisioning ---> active <---> suspended
--                     |
--                     +---------> failed
--
-- 'failed' ile 'inactive' ayri tutulur. Ilki kurulumun basarisiz oldugunu,
-- ikincisi calisan bir tenant'in kapatildigini anlatir. Ayni kolonda
-- birlestirmek, "kurulum tekrar denenebilir mi?" sorusunu cevapsiz birakirdi.
--
-- Basarisiz bir tenant icin yeni bir operasyon acilabilir; tenant 'failed'
-- kalirken yeni operasyon 'pending' baslar. Bu, operasyon durumunun tenant
-- durumundan ayri tutulmasinin gerekcesidir (bkz. migration 031).

ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_status_check;

ALTER TABLE tenants ADD CONSTRAINT tenants_status_check
    CHECK (status IN (
        'pending',       -- kayit olustu, kurulum henuz baslamadi
        'provisioning',  -- worker semayi hazirliyor
        'active',        -- istek kabul edilebilir
        'suspended',     -- gecici olarak durduruldu
        'inactive',      -- kapatildi
        'failed',        -- kurulum basarisiz oldu
        'trial'
    ));

COMMENT ON COLUMN tenants.status IS
    'Yasam dongusu durumu. Yalnizca active durumdaki tenant istek kabul eder.';
