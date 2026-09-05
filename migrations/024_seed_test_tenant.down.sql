-- Migration: Rollback Test Tenant
-- Description: Remove Can Akyüz test tenant and all related data
-- Author: Keystone Team
-- Date: 2025-10-05

-- Delete sample student data
DELETE FROM students WHERE tenant_id = 'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID;

-- Delete tenant tool activations
DELETE FROM tenant_tools WHERE tenant_id = 'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID;

-- Delete tenant module activations
DELETE FROM tenant_modules WHERE tenant_id = 'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID;

-- Delete test tenant (CASCADE will handle remaining dependencies)
DELETE FROM tenants WHERE id = 'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID;

-- Success message
DO $$
BEGIN
    RAISE NOTICE 'Test tenant removed successfully!';
END $$;
