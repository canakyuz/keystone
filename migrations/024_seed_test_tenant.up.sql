-- Migration: Create Test Tenant with Module Activations
-- Description: Create Can Akyüz test tenant and activate LMS + Payment modules
-- Author: NexSpaces Team
-- Date: 2025-10-05

-- ============================================================================
-- PART 1: CREATE TEST TENANT (Can Akyüz)
-- ============================================================================

-- Insert test tenant
INSERT INTO tenants (
    id,
    name,
    slug,
    email,
    phone,
    status,
    plan,
    subscription_start,
    trial_ends_at,
    custom_domain,
    custom_domain_verified,
    settings,
    metadata
) VALUES (
    'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID, -- Fixed UUID for test tenant
    'Can Akyüz Tech Academy',
    'canakyuz',
    'can@akyuz.tech',
    '+90 555 123 4567',
    'trial',
    'pro',
    NOW(),
    NOW() + INTERVAL '30 days',
    NULL, -- No custom domain yet
    FALSE,
    '{"timezone": "Europe/Istanbul", "language": "tr", "currency": "TRY"}'::jsonb,
    '{"company": "Can Akyüz Tech", "industry": "Education", "country": "Turkey"}'::jsonb
) ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- PART 2: ACTIVATE LMS MODULE
-- ============================================================================

INSERT INTO tenant_modules (
    id,
    tenant_id,
    module_id,
    status,
    is_enabled,
    installed_at,
    activated_at,
    installed_version,
    configuration,
    features_enabled,
    limits,
    current_usage,
    subscription_status,
    subscription_start,
    trial_ends_at,
    pricing_plan,
    billing_cycle,
    setup_completed,
    onboarding_completed,
    allowed_roles,
    notes
)
SELECT
    gen_random_uuid(),
    'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID, -- Can Akyüz tenant
    m.id,
    'active',
    TRUE,
    NOW(),
    NOW(),
    '1.0.0',
    '{
        "theme": "professional",
        "language": "tr",
        "timezone": "Europe/Istanbul",
        "email_notifications": true,
        "lesson_reminders": true,
        "auto_attendance": false
    }'::jsonb,
    '["lessons", "students", "assignments", "performance_tracking", "materials"]'::jsonb,
    '{
        "max_lessons": 500,
        "max_students": 200,
        "max_assignments": 1000,
        "max_materials_per_lesson": 20
    }'::jsonb,
    '{
        "lessons_count": 0,
        "students_count": 0,
        "assignments_count": 0,
        "active_lessons": 0
    }'::jsonb,
    'trial',
    NOW(),
    NOW() + INTERVAL '30 days',
    'pro',
    'monthly',
    TRUE,
    TRUE,
    ARRAY['admin', 'teacher', 'student'],
    'Initial LMS module activation for test tenant'
FROM modules m
WHERE m.code = 'LMS'
ON CONFLICT (tenant_id, module_id) DO NOTHING;

-- ============================================================================
-- PART 3: ACTIVATE PAYMENT TOOL
-- ============================================================================

INSERT INTO tenant_tools (
    id,
    tenant_id,
    tool_id,
    module_id,
    status,
    is_enabled,
    installed_at,
    activated_at,
    installed_version,
    configuration,
    api_keys,
    webhook_config,
    integration_enabled,
    integration_status,
    integration_verified,
    limits,
    current_usage,
    rate_limits,
    subscription_status,
    subscription_start,
    trial_ends_at,
    pricing_plan,
    billing_cycle,
    setup_completed,
    onboarding_completed,
    allowed_roles,
    health_status,
    notes
)
SELECT
    gen_random_uuid(),
    'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID, -- Can Akyüz tenant
    t.id,
    tm.module_id, -- Linked to LMS module
    'active',
    TRUE,
    NOW(),
    NOW(),
    '1.0.0',
    '{
        "default_provider": "stripe",
        "currency": "TRY",
        "allow_installments": false,
        "capture_method": "automatic",
        "statement_descriptor": "CAN AKYUZ TECH"
    }'::jsonb,
    '{
        "stripe_public_key": "pk_test_...",
        "stripe_secret_key": "sk_test_...",
        "iyzico_api_key": null,
        "iyzico_secret_key": null
    }'::jsonb, -- Note: In production, these should be encrypted
    '{
        "payment_success_url": "https://canakyuz.nexpaces.com/payment/success",
        "payment_cancel_url": "https://canakyuz.nexpaces.com/payment/cancel",
        "webhook_url": "https://canakyuz.nexpaces.com/api/webhooks/payment",
        "webhook_secret": "whsec_test_..."
    }'::jsonb,
    TRUE,
    'connected',
    TRUE,
    '{
        "max_transactions_per_month": 1000,
        "max_amount_per_transaction": 10000
    }'::jsonb,
    '{
        "transactions_count": 0,
        "total_amount_collected": 0,
        "last_transaction_at": null
    }'::jsonb,
    '{
        "requests_per_minute": 60
    }'::jsonb,
    'trial',
    NOW(),
    NOW() + INTERVAL '30 days',
    'pro',
    'per_transaction',
    TRUE,
    TRUE,
    ARRAY['admin', 'teacher'],
    'healthy',
    'Payment tool activated for LMS lesson payments'
FROM tools t
CROSS JOIN tenant_modules tm
WHERE t.code = 'PAYMENT'
  AND tm.tenant_id = 'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID
  AND tm.module_id = (SELECT id FROM modules WHERE code = 'LMS')
ON CONFLICT (tenant_id, tool_id) DO NOTHING;

-- ============================================================================
-- PART 4: ACTIVATE WEBHOOK TOOL (Recommended with Payment)
-- ============================================================================

INSERT INTO tenant_tools (
    id,
    tenant_id,
    tool_id,
    module_id,
    status,
    is_enabled,
    installed_at,
    activated_at,
    installed_version,
    configuration,
    webhook_config,
    integration_enabled,
    integration_status,
    limits,
    current_usage,
    rate_limits,
    subscription_status,
    pricing_plan,
    setup_completed,
    onboarding_completed,
    allowed_roles,
    health_status,
    notes
)
SELECT
    gen_random_uuid(),
    'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID, -- Can Akyüz tenant
    t.id,
    NULL, -- Global tool, not module-specific
    'active',
    TRUE,
    NOW(),
    NOW(),
    '1.0.0',
    '{
        "retry_attempts": 3,
        "retry_delay_seconds": 60,
        "signature_verification": true,
        "log_all_events": true
    }'::jsonb,
    '{
        "endpoint": "https://canakyuz.nexpaces.com/api/webhooks/handler",
        "events": ["payment.*", "refund.*", "3ds.*"],
        "headers": {
            "X-Tenant-ID": "aaaaaaaa-bbbb-cccc-dddd-000000000001"
        }
    }'::jsonb,
    TRUE,
    'connected',
    '{
        "max_events_per_day": 10000
    }'::jsonb,
    '{
        "events_received": 0,
        "events_processed": 0,
        "events_failed": 0
    }'::jsonb,
    '{
        "events_per_second": 100
    }'::jsonb,
    'trial',
    'free',
    TRUE,
    TRUE,
    ARRAY['admin'],
    'healthy',
    'Webhook tool for payment event notifications'
FROM tools t
WHERE t.code = 'WEBHOOK'
ON CONFLICT (tenant_id, tool_id) DO NOTHING;

-- ============================================================================
-- PART 5: CREATE SAMPLE STUDENT (for LMS demo)
-- ============================================================================

-- Note: This requires students table to exist
-- Adding sample data for demonstration purposes

INSERT INTO students (
    id,
    tenant_id,
    first_name,
    last_name,
    email,
    phone,
    grade_level,
    status,
    metadata,
    created_at,
    updated_at
)
VALUES (
    gen_random_uuid(),
    'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID,
    'Ahmet',
    'Yılmaz',
    'ahmet.yilmaz@example.com',
    '+90 555 999 8888',
    '10',
    'active',
    '{"parent_name": "Mehmet Yılmaz", "parent_phone": "+90 555 999 7777", "emergency_contact": "+90 555 999 6666"}'::jsonb,
    NOW(),
    NOW()
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- Success message
-- ============================================================================

DO $$
DECLARE
    tenant_count INTEGER;
    module_count INTEGER;
    tool_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO tenant_count FROM tenants WHERE id = 'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID;
    SELECT COUNT(*) INTO module_count FROM tenant_modules WHERE tenant_id = 'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID;
    SELECT COUNT(*) INTO tool_count FROM tenant_tools WHERE tenant_id = 'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID;

    RAISE NOTICE '';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Test Tenant Created Successfully!';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Tenant: Can Akyüz Tech Academy';
    RAISE NOTICE 'Slug: canakyuz';
    RAISE NOTICE 'Email: can@akyuz.tech';
    RAISE NOTICE 'URL: https://canakyuz.nexpaces.com';
    RAISE NOTICE '------------------------------------------------';
    RAISE NOTICE 'Activated Modules: %', module_count;
    RAISE NOTICE '  - LMS (Learning Management System)';
    RAISE NOTICE 'Activated Tools: %', tool_count;
    RAISE NOTICE '  - Payment (Stripe)';
    RAISE NOTICE '  - Webhook';
    RAISE NOTICE '------------------------------------------------';
    RAISE NOTICE 'Status: Trial (30 days)';
    RAISE NOTICE 'Plan: Pro';
    RAISE NOTICE '================================================';
END $$;
