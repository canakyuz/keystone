-- Migration: Seed development/staging tenant
-- Purpose : Populate a sample tenant with preconfigured modules/tools for testing locally.
-- CAUTION : Not intended for production environments. Guarded to run safely multiple times.
-- WARNING : SKIPS OUTSIDE development/staging/test environments

DO $$
DECLARE
    v_tenant_id UUID := 'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID;
    tenant_exists BOOLEAN;
    current_env TEXT;
BEGIN
    -- ⚠️ PRODUCTION GUARD: Prevent execution in production
    current_env := COALESCE(current_setting('app.environment', true), 'production');

    IF current_env NOT IN ('development', 'staging', 'test') THEN
        RAISE NOTICE 'Skipping test tenant seed in % environment', current_env;
        RETURN;
    END IF;

    RAISE NOTICE 'Running in % environment - proceeding with test tenant seed', current_env;

    SELECT EXISTS(SELECT 1 FROM tenants WHERE id = v_tenant_id) INTO tenant_exists;

    IF tenant_exists THEN
        RAISE NOTICE 'Test tenant already exists. Skipping seed.';
        RETURN;
    END IF;

    -- =====================================================================
    -- Tenant record
    -- =====================================================================
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
        v_tenant_id,
        'Can Akyüz Tech Academy',
        'canakyuz',
        'can@akyuz.tech',
        '+90 555 123 4567',
        'trial',
        'pro',
        NOW(),
        NOW() + INTERVAL '30 days',
        NULL,
        FALSE,
        '{"timezone": "Europe/Istanbul", "language": "tr", "currency": "TRY"}'::jsonb,
        '{"company": "Can Akyüz Tech", "industry": "Education", "country": "Turkey"}'::jsonb
    ) ON CONFLICT (id) DO NOTHING;

    -- =====================================================================
    -- Module activations
    -- =====================================================================
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
        v_tenant_id,
        m.id,
        'active',
        TRUE,
        NOW(),
        NOW(),
        '1.0.0',
        '{"theme": "professional", "language": "tr", "timezone": "Europe/Istanbul", "email_notifications": true, "lesson_reminders": true, "auto_attendance": false}'::jsonb,
        '["lessons", "students", "assignments", "performance_tracking", "materials"]'::jsonb,
        '{"max_lessons": 500, "max_students": 200, "max_assignments": 1000, "max_materials_per_lesson": 20}'::jsonb,
        '{"lessons_count": 0, "students_count": 0, "assignments_count": 0, "active_lessons": 0}'::jsonb,
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

    -- =====================================================================
    -- Tool activations
    -- =====================================================================
    INSERT INTO tenant_tools (
        id,
        tenant_id,
        tool_id,
        module_id,
        status,
        is_enabled,
        integration_status,
        health_status,
        installed_at,
        activated_at,
        configuration,
        usage,
        subscription_status,
        subscription_start,
        trial_ends_at,
        auto_renew,
        requires_setup,
        requires_webhook,
        notes
    )
    SELECT
        gen_random_uuid(),
        v_tenant_id,
        t.id,
        NULL,
        'active',
        TRUE,
        'verified',
        'healthy',
        NOW(),
        NOW(),
        '{"provider": "stripe", "api_key": "REPLACE_WITH_STRIPE_TEST_KEY", "currency": "TRY", "three_ds": true}'::jsonb,
        '{"transactions": 0, "volume": 0}'::jsonb,
        'trial',
        NOW(),
        NOW() + INTERVAL '30 days',
        TRUE,
        FALSE,
        TRUE,
        'Initial payment tool activation'
    FROM tools t
    WHERE t.code = 'PAYMENT'
    ON CONFLICT (tenant_id, tool_id) DO NOTHING;

    INSERT INTO tenant_tools (
        id,
        tenant_id,
        tool_id,
        module_id,
        status,
        is_enabled,
        integration_status,
        health_status,
        installed_at,
        activated_at,
        configuration,
        usage,
        subscription_status,
        subscription_start,
        trial_ends_at,
        auto_renew,
        requires_setup,
        requires_webhook,
        notes
    )
    SELECT
        gen_random_uuid(),
        v_tenant_id,
        t.id,
        NULL,
        'active',
        TRUE,
        'verified',
        'healthy',
        NOW(),
        NOW(),
        '{"endpoint": "https://api.acme.dev/webhooks/payments", "secret": "REPLACE_WITH_WEBHOOK_SECRET"}'::jsonb,
        '{"events_processed": 0}'::jsonb,
        'trial',
        NOW(),
        NOW() + INTERVAL '30 days',
        TRUE,
        FALSE,
        TRUE,
        'Webhook listener for payment events'
    FROM tools t
    WHERE t.code = 'WEBHOOK'
    ON CONFLICT (tenant_id, tool_id) DO NOTHING;

    -- =====================================================================
    -- Sample student data
    -- =====================================================================
    INSERT INTO students (
        id,
        tenant_id,
        first_name,
        last_name,
        email,
        phone,
        status,
        metadata,
        created_at,
        updated_at
    ) VALUES (
        gen_random_uuid(),
        v_tenant_id,
        'Ahmet',
        'Yılmaz',
        'ahmet.yilmaz@example.com',
        '+90 555 999 8888',
        'active',
        '{"parent_name": "Mehmet Yılmaz", "parent_phone": "+90 555 999 7777", "emergency_contact": "+90 555 999 6666"}'::jsonb,
        NOW(),
        NOW()
    ) ON CONFLICT DO NOTHING;

    -- =====================================================================
    -- Success message
    -- =====================================================================
    RAISE NOTICE '';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Test Tenant Created Successfully!';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Tenant: Can Akyüz Tech Academy';
    RAISE NOTICE 'Slug: canakyuz';
    RAISE NOTICE 'Email: can@akyuz.tech';
    RAISE NOTICE 'URL: https://canakyuz.nexpaces.com';
    RAISE NOTICE '------------------------------------------------';
    RAISE NOTICE 'Activated Modules: 1';
    RAISE NOTICE '  - LMS (Learning Management System)';
    RAISE NOTICE 'Activated Tools: 2';
    RAISE NOTICE '  - Payment (Stripe)';
    RAISE NOTICE '  - Webhook';
    RAISE NOTICE '------------------------------------------------';
    RAISE NOTICE 'Status: Trial (30 days)';
    RAISE NOTICE 'Plan: Pro';
    RAISE NOTICE '================================================';
END $$;
