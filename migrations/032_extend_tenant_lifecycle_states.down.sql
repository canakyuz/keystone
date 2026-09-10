-- 032 geri alma.
-- WARNING: the constraint cannot be added while tenants sit in the new states.
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_status_check;

ALTER TABLE tenants ADD CONSTRAINT tenants_status_check
    CHECK (status IN ('active', 'suspended', 'inactive', 'trial'));
