-- Migration: Rollback Registry Seed Data
-- Description: Remove all seeded modules, tools, and dependencies
-- Author: NexSpaces Team
-- Date: 2025-10-05

-- Delete in reverse order to respect foreign key constraints

-- Delete tool dependencies
DELETE FROM tool_dependencies WHERE tool_id IN (
    SELECT id FROM tools WHERE code IN ('REFUND', 'PAYMENT', 'WEBHOOK')
);

-- Delete module dependencies
DELETE FROM module_dependencies WHERE module_id IN (
    SELECT id FROM modules WHERE code IN ('LMS', 'CMS', 'PROJECT', 'BOOKING', 'SERVICE', 'WEBSITE')
);

-- Delete tools
DELETE FROM tools WHERE code IN ('PAYMENT', 'REFUND', 'WEBHOOK');

-- Delete modules
DELETE FROM modules WHERE code IN ('LMS', 'CMS', 'PROJECT', 'BOOKING', 'SERVICE', 'WEBSITE');

-- Success message
DO $$
BEGIN
    RAISE NOTICE 'Registry seed data removed successfully!';
END $$;
