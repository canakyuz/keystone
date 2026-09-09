-- Development Seed Script
-- Creates founder tenant, founder owner, and sector-specific users+modules
--
-- 🎓 PRODUCTION GUARD
-- Bu script SADECE development/staging ortamlarında çalışmalıdır.
-- Production'da çalıştırılması veri güvenliği riski oluşturur.

-- ⚠️ GUARD: Production environment check
DO $$
DECLARE
    v_environment TEXT;
BEGIN
    -- Check if running in production
    v_environment := current_setting('app.environment', true);

    IF v_environment = 'production' THEN
        RAISE EXCEPTION 'SEED GUARD: Development seed script cannot run in production environment. Set app.environment to development/staging first.';
    END IF;

    -- Also check database name as secondary guard
    IF current_database() LIKE '%prod%' OR current_database() LIKE '%production%' THEN
        RAISE EXCEPTION 'SEED GUARD: Cannot seed production database (database name contains "prod" or "production")';
    END IF;

    RAISE NOTICE 'SEED GUARD: Environment check passed (environment: %, database: %)', v_environment, current_database();
END $$;

-- 1. Founder tenant
DO $$
DECLARE
    v_tenant_id UUID := '550e8400-e29b-41d4-a716-446655440000'::UUID;
    v_schema TEXT;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = v_tenant_id) THEN
        INSERT INTO tenants (id, name, slug, email, status, plan, settings, metadata)
        VALUES (
            v_tenant_id,
            'Dev Academy',
            'canakyuz',
            'owner@dev.keystone.local',
            'active',
            'enterprise',
            '{"timezone":"Europe/Istanbul","language":"tr","currency":"TRY"}'::jsonb,
            '{"owner":"Dev Owner","industry":"Education"}'::jsonb
        ) ON CONFLICT (id) DO NOTHING;
    END IF;

    SELECT schema_name INTO v_schema FROM tenants WHERE id = v_tenant_id;
    IF v_schema IS NOT NULL THEN
        EXECUTE format('CREATE SCHEMA IF NOT EXISTS %I', v_schema);
    END IF;
END $$;

-- 2. Founder owner account
INSERT INTO users (
    id, tenant_id, email, password, first_name, last_name,
    role, status, email_verified, email_verified_at, metadata
) VALUES (
    uuid_generate_v4(),
    '550e8400-e29b-41d4-a716-446655440000',
    'owner@dev.keystone.local',
    crypt('DevPass123!', gen_salt('bf')),
    'Dev',
    'Owner',
    'owner',
    'active',
    TRUE,
    NOW(),
    '{"seed":"founder","environment":"development"}'::jsonb
) ON CONFLICT (email, tenant_id) DO NOTHING;

-- 3. Development test tenant with sector accounts
DO $$
DECLARE
    v_tenant_id UUID := 'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID;
    v_schema TEXT;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = v_tenant_id) THEN
        INSERT INTO tenants (
            id, name, slug, email, phone, status, plan,
            subscription_start, trial_ends_at, settings, metadata
        ) VALUES (
            v_tenant_id,
            'Can Akyüz Tech Academy (Dev)',
            'canakyuz-dev',
            'can@akyuz.tech',
            '+90 555 123 4567',
            'trial',
            'pro',
            NOW(),
            NOW() + INTERVAL '30 days',
            '{"timezone":"Europe/Istanbul","language":"tr","currency":"TRY"}'::jsonb,
            '{"company":"Can Akyüz Tech","industry":"Education","country":"Turkey"}'::jsonb
        ) ON CONFLICT (id) DO NOTHING;
    END IF;

    SELECT schema_name INTO v_schema FROM tenants WHERE id = v_tenant_id;
    IF v_schema IS NOT NULL THEN
        EXECUTE format('CREATE SCHEMA IF NOT EXISTS %I', v_schema);
    END IF;
END $$;

-- Activate LMS module for dev tenant (if registry prepared)
INSERT INTO tenant_modules (
    id, tenant_id, module_id, status, is_enabled,
    installed_at, activated_at, installed_version,
    configuration, features_enabled, limits, current_usage,
    subscription_status, subscription_start, trial_ends_at,
    pricing_plan, billing_cycle, setup_completed, onboarding_completed, allowed_roles, notes
)
SELECT
    gen_random_uuid(),
    'aaaaaaaa-bbbb-cccc-dddd-000000000001',
    m.id,
    'active',
    TRUE,
    NOW(),
    NOW(),
    '1.0.0',
    '{"language":"tr","timezone":"Europe/Istanbul"}'::jsonb,
    '["lessons","students","assignments"]'::jsonb,
    '{"max_lessons": 200, "max_students": 200}'::jsonb,
    '{"lessons_count":0}'::jsonb,
    'trial',
    NOW(),
    NOW() + INTERVAL '30 days',
    'pro',
    'monthly',
    TRUE,
    TRUE,
    ARRAY['admin','teacher','student'],
    'Dev seed LMS module'
FROM modules m
WHERE m.code = 'LMS'
ON CONFLICT (tenant_id, module_id) DO NOTHING;

-- Sector-specific users for dev tenant
INSERT INTO users (
    id, tenant_id, email, password, first_name, last_name,
    role, status, email_verified, email_verified_at, metadata
) VALUES
    (gen_random_uuid(), 'aaaaaaaa-bbbb-cccc-dddd-000000000001', 'education.admin@dev.keystone.dev', crypt('DevPass123!', gen_salt('bf')), 'Eğitim', 'Yöneticisi', 'admin', 'active', TRUE, NOW(), '{"sector":"education","role":"admin"}'::jsonb),
    (gen_random_uuid(), 'aaaaaaaa-bbbb-cccc-dddd-000000000001', 'education.instructor@dev.keystone.dev', crypt('DevPass123!', gen_salt('bf')), 'Eğitim', 'Eğitmeni', 'editor', 'active', TRUE, NOW(), '{"sector":"education","role":"instructor"}'::jsonb),
    (gen_random_uuid(), 'aaaaaaaa-bbbb-cccc-dddd-000000000001', 'education.observer@dev.keystone.dev', crypt('DevPass123!', gen_salt('bf')), 'Eğitim', 'Gözlemci', 'viewer', 'active', TRUE, NOW(), '{"sector":"education","role":"observer"}'::jsonb),
    (gen_random_uuid(), 'aaaaaaaa-bbbb-cccc-dddd-000000000001', 'hospitality.manager@dev.keystone.dev', crypt('DevPass123!', gen_salt('bf')), 'Hospitality', 'Manager', 'admin', 'active', TRUE, NOW(), '{"sector":"hospitality","role":"manager"}'::jsonb),
    (gen_random_uuid(), 'aaaaaaaa-bbbb-cccc-dddd-000000000001', 'hospitality.frontdesk@dev.keystone.dev', crypt('DevPass123!', gen_salt('bf')), 'Hospitality', 'Frontdesk', 'editor', 'active', TRUE, NOW(), '{"sector":"hospitality","role":"frontdesk"}'::jsonb),
    (gen_random_uuid(), 'aaaaaaaa-bbbb-cccc-dddd-000000000001', 'commerce.manager@dev.keystone.dev', crypt('DevPass123!', gen_salt('bf')), 'E-Ticaret', 'Yöneticisi', 'admin', 'active', TRUE, NOW(), '{"sector":"commerce","role":"manager"}'::jsonb),
    (gen_random_uuid(), 'aaaaaaaa-bbbb-cccc-dddd-000000000001', 'commerce.support@dev.keystone.dev', crypt('DevPass123!', gen_salt('bf')), 'E-Ticaret', 'Destek', 'viewer', 'active', TRUE, NOW(), '{"sector":"commerce","role":"support"}'::jsonb),
    (gen_random_uuid(), 'aaaaaaaa-bbbb-cccc-dddd-000000000001', 'content.editor@dev.keystone.dev', crypt('DevPass123!', gen_salt('bf')), 'İçerik', 'Editörü', 'editor', 'active', TRUE, NOW(), '{"sector":"content","role":"editor"}'::jsonb),
    (gen_random_uuid(), 'aaaaaaaa-bbbb-cccc-dddd-000000000001', 'crm.lead@dev.keystone.dev', crypt('DevPass123!', gen_salt('bf')), 'CRM', 'Satış', 'admin', 'active', TRUE, NOW(), '{"sector":"crm","role":"sales"}'::jsonb),
    (gen_random_uuid(), 'aaaaaaaa-bbbb-cccc-dddd-000000000001', 'oms.dispatch@dev.keystone.dev', crypt('DevPass123!', gen_salt('bf')), 'Operasyon', 'Dispatch', 'admin', 'active', TRUE, NOW(), '{"sector":"oms","role":"dispatch"}'::jsonb),
    (gen_random_uuid(), 'aaaaaaaa-bbbb-cccc-dddd-000000000001', 'erp.operations@dev.keystone.dev', crypt('DevPass123!', gen_salt('bf')), 'ERP', 'Operasyon', 'admin', 'active', TRUE, NOW(), '{"sector":"erp","role":"operations"}'::jsonb)
ON CONFLICT (email, tenant_id) DO NOTHING;

-- Sample student data for dev tenant
INSERT INTO students (
    id, tenant_id, first_name, last_name, email, phone, status,
    level, grade, school, parent_name, parent_email, parent_phone,
    metadata, created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'aaaaaaaa-bbbb-cccc-dddd-000000000001',
    'Ahmet',
    'Yılmaz',
    'ahmet.yilmaz@example.com',
    '+90 555 999 8888',
    'active',
    'high_school',
    '10',
    'Dev Academy',
    'Mehmet Yılmaz',
    'mehmet.yilmaz@example.com',
    '+90 555 999 7777',
    '{"emergency_contact": "+90 555 999 6666"}'::jsonb,
    NOW(),
    NOW()
) ON CONFLICT (tenant_id, email) DO NOTHING;

