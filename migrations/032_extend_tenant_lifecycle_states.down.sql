-- 032 geri alma.
-- DIKKAT: yeni durumlardaki tenant'lar varsa kisit eklenemez.
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_status_check;

ALTER TABLE tenants ADD CONSTRAINT tenants_status_check
    CHECK (status IN ('active', 'suspended', 'inactive', 'trial'));
