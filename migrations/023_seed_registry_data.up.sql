-- Migration: Seed Registry Data
-- Description: Populate modules, tools, and dependencies for Keystone platform
-- Author: Keystone Team
-- Date: 2025-10-05

-- ============================================================================
-- PART 1: MODULES (Catalog)
-- ============================================================================

-- 1.1 LMS (Learning Management System) Module
INSERT INTO modules (
    id,
    name,
    slug,
    code,
    display_name,
    description,
    category,
    module_type,
    status,
    is_public,
    version,
    pricing_model,
    base_price,
    currency,
    features,
    capabilities,
    database_tables,
    default_limits,
    icon,
    tags
) VALUES (
    gen_random_uuid(),
    'Learning Management System',
    'lms',
    'LMS',
    'LMS - Learning Management',
    'Complete learning management system with lessons, students, assignments, and performance tracking. Perfect for tutoring, online courses, and educational institutions.',
    'education',
    'standard',
    'active',
    TRUE,
    '1.0.0',
    'subscription',
    29.99,
    'USD',
    '["One-on-one and group lessons", "Online and in-person sessions", "Assignment management", "Performance tracking", "Attendance tracking", "Material/resource management", "Homework tracking", "Progress reports", "Meeting URL integration (Zoom/Meet)"]'::jsonb,
    '{"lesson_types": ["one-on-one", "group", "online", "in-person"], "supports_video": true, "supports_materials": true, "supports_assignments": true}'::jsonb,
    ARRAY['lessons', 'students', 'assignments'],
    '{"max_lessons": 100, "max_students": 50, "max_assignments": 200}'::jsonb,
    'graduation-cap',
    ARRAY['education', 'learning', 'tutoring', 'courses', 'lessons', 'students']
) ON CONFLICT (name) DO NOTHING;

-- 1.2 CMS/Blog Module
INSERT INTO modules (
    id,
    name,
    slug,
    code,
    display_name,
    description,
    category,
    module_type,
    status,
    is_public,
    version,
    pricing_model,
    base_price,
    currency,
    features,
    capabilities,
    database_tables,
    default_limits,
    icon,
    tags
) VALUES (
    gen_random_uuid(),
    'Content Management System',
    'cms',
    'CMS',
    'CMS - Content & Blog',
    'Powerful content management system with blog capabilities. Create, publish, and manage articles, posts, and categories. Includes featured content, tags, SEO optimization, and view tracking.',
    'content',
    'standard',
    'active',
    TRUE,
    '1.0.0',
    'free',
    0.00,
    'USD',
    '["Blog post creation", "Category management", "Draft/Publish workflow", "Featured content", "Tag system", "View count tracking", "Rich text editor", "SEO optimization", "Image support"]'::jsonb,
    '{"supports_categories": true, "supports_tags": true, "supports_seo": true, "supports_drafts": true}'::jsonb,
    ARRAY['blog_posts', 'blog_categories'],
    '{"max_posts": 1000, "max_categories": 50}'::jsonb,
    'newspaper',
    ARRAY['blog', 'content', 'cms', 'publishing', 'articles', 'writing']
) ON CONFLICT (name) DO NOTHING;

-- 1.3 Project Management Module
INSERT INTO modules (
    id,
    name,
    slug,
    code,
    display_name,
    description,
    category,
    module_type,
    status,
    is_public,
    version,
    pricing_model,
    base_price,
    currency,
    features,
    capabilities,
    database_tables,
    default_limits,
    icon,
    tags
) VALUES (
    gen_random_uuid(),
    'Project Management',
    'project-management',
    'PROJECT',
    'Project Management',
    'Portfolio and project management module. Showcase your projects, track progress, manage client work, and display your portfolio. Perfect for freelancers, agencies, and developers.',
    'management',
    'standard',
    'active',
    TRUE,
    '1.0.0',
    'free',
    0.00,
    'USD',
    '["Project tracking", "Status management (Planned/In-Progress/Completed)", "Client tracking", "Technology stack display", "Image gallery", "Live URL & GitHub links", "Featured projects", "View count", "Project categories"]'::jsonb,
    '{"supports_portfolio": true, "supports_client_tracking": true, "supports_gallery": true, "supports_github": true}'::jsonb,
    ARRAY['projects'],
    '{"max_projects": 100, "max_images_per_project": 10}'::jsonb,
    'briefcase',
    ARRAY['portfolio', 'projects', 'freelance', 'agency', 'showcase']
) ON CONFLICT (name) DO NOTHING;

-- 1.4 Booking/Appointment Module
INSERT INTO modules (
    id,
    name,
    slug,
    code,
    display_name,
    description,
    category,
    module_type,
    status,
    is_public,
    version,
    pricing_model,
    base_price,
    currency,
    features,
    capabilities,
    database_tables,
    default_limits,
    icon,
    tags
) VALUES (
    gen_random_uuid(),
    'Booking & Appointments',
    'booking',
    'BOOKING',
    'Booking System',
    'Complete booking and appointment management system. Schedule appointments, manage availability, send reminders, and handle cancellations. Perfect for consultants, healthcare, and service providers.',
    'hospitality',
    'standard',
    'active',
    TRUE,
    '1.0.0',
    'subscription',
    19.99,
    'USD',
    '["Appointment scheduling", "Availability management", "Client information tracking", "Reminder system", "Cancellation management", "Meeting URL support", "Status tracking (Pending/Confirmed/Completed)", "No-show tracking", "Time slot management"]'::jsonb,
    '{"supports_reminders": true, "supports_availability": true, "supports_recurring": false, "supports_meeting_urls": true}'::jsonb,
    ARRAY['appointments', 'availabilities'],
    '{"max_appointments": 500, "max_availability_slots": 200}'::jsonb,
    'calendar-check',
    ARRAY['booking', 'appointments', 'scheduling', 'healthcare', 'consulting', 'reservations']
) ON CONFLICT (name) DO NOTHING;

-- 1.5 Service Catalog Module
INSERT INTO modules (
    id,
    name,
    slug,
    code,
    display_name,
    description,
    category,
    module_type,
    status,
    is_public,
    version,
    pricing_model,
    base_price,
    currency,
    features,
    capabilities,
    database_tables,
    default_limits,
    icon,
    tags
) VALUES (
    gen_random_uuid(),
    'Service Catalog',
    'service-catalog',
    'SERVICE',
    'Service & Product Catalog',
    'Service and product catalog module. Define your offerings, pricing models, and manage service tiers. Supports one-time, recurring, subscription, and usage-based pricing.',
    'commerce',
    'standard',
    'active',
    TRUE,
    '1.0.0',
    'free',
    0.00,
    'USD',
    '["Service/Product catalog", "Multiple pricing models", "Feature lists", "Public/Private visibility", "Featured services", "Image support", "Duration tracking", "Max client limits", "Category management"]'::jsonb,
    '{"pricing_models": ["one_time", "recurring", "subscription", "usage_based"], "supports_features": true, "supports_visibility": true}'::jsonb,
    ARRAY['services'],
    '{"max_services": 100}'::jsonb,
    'box',
    ARRAY['services', 'products', 'catalog', 'pricing', 'offerings', 'ecommerce']
) ON CONFLICT (name) DO NOTHING;

-- 1.6 Website Builder Module (Platform Core)
INSERT INTO modules (
    id,
    name,
    slug,
    code,
    display_name,
    description,
    category,
    module_type,
    status,
    is_public,
    version,
    pricing_model,
    base_price,
    currency,
    features,
    capabilities,
    database_tables,
    default_limits,
    icon,
    tags,
    metadata
) VALUES (
    gen_random_uuid(),
    'Website Builder',
    'website-builder',
    'WEBSITE',
    'Website Builder',
    'Multi-tenant website builder. Create and publish websites with custom domains, templates, and module integration. Core platform module that enables tenant website creation.',
    'other',
    'standard',
    'active',
    FALSE, -- Not visible in marketplace (platform infrastructure)
    '1.0.0',
    'free',
    0.00,
    'USD',
    '["Website creation", "Custom domain support", "Template system", "Multi-tenant isolation", "Draft/Published workflow", "Domain verification", "Website types (CMS/CRM/E-commerce/Education/Hospitality/ERP/Blog/Portfolio)"]'::jsonb,
    '{"supports_custom_domain": true, "supports_templates": true, "supports_multi_types": true}'::jsonb,
    ARRAY['websites'],
    '{"max_websites": 10}'::jsonb,
    'globe',
    ARRAY['website', 'builder', 'platform', 'multi-tenant'],
    '{"is_platform_core": true, "auto_enabled": true}'::jsonb
) ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- PART 2: TOOLS (Catalog)
-- ============================================================================

-- 2.1 Payment Tool
INSERT INTO tools (
    id,
    name,
    slug,
    code,
    display_name,
    description,
    category,
    tool_type,
    scope,
    status,
    is_public,
    version,
    pricing_model,
    base_price,
    currency,
    transaction_fee_percentage,
    transaction_fee_fixed,
    features,
    capabilities,
    integration_provider,
    integration_type,
    requires_api_keys,
    requires_webhook,
    database_tables,
    default_limits,
    rate_limits,
    icon,
    tags
) VALUES (
    gen_random_uuid(),
    'Payment Gateway',
    'payment',
    'PAYMENT',
    'Payment Gateway',
    'Multi-provider payment gateway tool. Supports Stripe, Iyzico, and Checkout.com. Includes 3DS authentication, installment support (Turkey), card management, and webhook integration.',
    'payment',
    'global',
    'cross_module',
    'active',
    TRUE,
    '1.0.0',
    'transaction_fee',
    0.00,
    'USD',
    2.50, -- 2.5% transaction fee
    0.30, -- $0.30 fixed fee per transaction
    '["Multi-provider support (Stripe, Iyzico, Checkout)", "3DS authentication (SCA compliant)", "Installment support (Turkey market)", "Card information (PCI-compliant)", "Customer information tracking", "Billing address", "Payment status tracking", "Webhook integration", "Metadata support"]'::jsonb,
    '{"providers": ["stripe", "iyzico", "checkout"], "supports_3ds": true, "supports_installments": true, "pci_compliant": true}'::jsonb,
    'Stripe, Iyzico, Checkout.com',
    'third_party',
    TRUE, -- Requires API keys
    TRUE, -- Requires webhook
    ARRAY['payments'],
    '{"max_transactions_per_month": 1000}'::jsonb,
    '{"requests_per_minute": 60}'::jsonb,
    'credit-card',
    ARRAY['payment', 'stripe', 'iyzico', 'checkout', 'gateway', 'transaction', 'billing']
) ON CONFLICT (name) DO NOTHING;

-- 2.2 Refund Tool
INSERT INTO tools (
    id,
    name,
    slug,
    code,
    display_name,
    description,
    category,
    tool_type,
    scope,
    status,
    is_public,
    version,
    pricing_model,
    base_price,
    currency,
    features,
    capabilities,
    integration_provider,
    integration_type,
    requires_api_keys,
    requires_webhook,
    database_tables,
    default_limits,
    icon,
    tags
) VALUES (
    gen_random_uuid(),
    'Refund Management',
    'refund',
    'REFUND',
    'Refund Tool',
    'Refund and return management tool. Process full or partial refunds, track refund status, and manage refund reasons. Works with Payment tool for seamless refund processing.',
    'payment',
    'global',
    'cross_module',
    'active',
    TRUE,
    '1.0.0',
    'free',
    0.00,
    'USD',
    '["Full/Partial refund support", "Multi-provider support", "Refund status tracking", "Refund reasons (Customer request, Duplicate, Fraudulent, Other)", "Provider integration", "Audit trail"]'::jsonb,
    '{"supports_partial_refund": true, "supports_full_refund": true, "audit_enabled": true}'::jsonb,
    'Stripe, Iyzico, Checkout.com',
    'third_party',
    FALSE, -- Uses parent Payment tool API keys
    FALSE,
    ARRAY['refunds'],
    '{"max_refunds_per_month": 100}'::jsonb,
    'rotate-ccw',
    ARRAY['refund', 'return', 'payment', 'transaction', 'reversal']
) ON CONFLICT (name) DO NOTHING;

-- 2.3 Webhook Tool
INSERT INTO tools (
    id,
    name,
    slug,
    code,
    display_name,
    description,
    category,
    tool_type,
    scope,
    status,
    is_public,
    version,
    pricing_model,
    base_price,
    currency,
    features,
    capabilities,
    integration_type,
    requires_webhook,
    database_tables,
    default_limits,
    rate_limits,
    icon,
    tags
) VALUES (
    gen_random_uuid(),
    'Webhook Handler',
    'webhook',
    'WEBHOOK',
    'Webhook Events',
    'Provider-agnostic webhook event handler. Processes payment, refund, 3DS, and chargeback events. Includes idempotency, signature validation, and retry mechanism.',
    'integration',
    'global',
    'cross_module',
    'active',
    TRUE,
    '1.0.0',
    'free',
    0.00,
    'USD',
    '["Provider-agnostic events", "Idempotency support", "Signature validation", "Retry mechanism", "Processing status tracking", "Request metadata (IP, headers)", "Event correlation", "Support for payment/refund/3DS/chargeback events"]'::jsonb,
    '{"event_types": ["payment.*", "refund.*", "3ds.*", "chargeback.*"], "supports_idempotency": true, "supports_retry": true}'::jsonb,
    'native',
    TRUE,
    ARRAY['payment_events'],
    '{"max_events_per_day": 10000}'::jsonb,
    '{"events_per_second": 100}'::jsonb,
    'zap',
    ARRAY['webhook', 'events', 'integration', 'notification', 'automation']
) ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- PART 3: MODULE DEPENDENCIES
-- ============================================================================

-- 3.1 LMS Module Dependencies
-- LMS optionally uses Payment tool (for paid lessons)
INSERT INTO module_dependencies (
    module_id,
    depends_on_tool_id,
    dependency_type,
    dependency_scope,
    auto_install,
    priority,
    description
)
SELECT
    m.id,
    t.id,
    'optional',
    'feature',
    FALSE,
    5,
    'Enable payment collection for lessons and courses'
FROM modules m
CROSS JOIN tools t
WHERE m.code = 'LMS'
  AND t.code = 'PAYMENT'
ON CONFLICT DO NOTHING;

-- 3.2 Booking Module Dependencies
-- Booking optionally uses Payment tool (for appointment deposits/payments)
INSERT INTO module_dependencies (
    module_id,
    depends_on_tool_id,
    dependency_type,
    dependency_scope,
    auto_install,
    priority,
    description
)
SELECT
    m.id,
    t.id,
    'optional',
    'feature',
    FALSE,
    5,
    'Enable payment collection for appointments and reservations'
FROM modules m
CROSS JOIN tools t
WHERE m.code = 'BOOKING'
  AND t.code = 'PAYMENT'
ON CONFLICT DO NOTHING;

-- 3.3 Service Catalog Module Dependencies
-- Service module optionally uses Payment tool (for service purchases)
INSERT INTO module_dependencies (
    module_id,
    depends_on_tool_id,
    dependency_type,
    dependency_scope,
    auto_install,
    priority,
    description
)
SELECT
    m.id,
    t.id,
    'optional',
    'runtime',
    FALSE,
    10,
    'Enable service/product purchases and checkout'
FROM modules m
CROSS JOIN tools t
WHERE m.code = 'SERVICE'
  AND t.code = 'PAYMENT'
ON CONFLICT DO NOTHING;

-- Service module optionally uses Refund tool (for service refunds)
INSERT INTO module_dependencies (
    module_id,
    depends_on_tool_id,
    dependency_type,
    dependency_scope,
    auto_install,
    priority,
    description
)
SELECT
    m.id,
    t.id,
    'optional',
    'runtime',
    FALSE,
    5,
    'Enable refund processing for services and products'
FROM modules m
CROSS JOIN tools t
WHERE m.code = 'SERVICE'
  AND t.code = 'REFUND'
ON CONFLICT DO NOTHING;

-- ============================================================================
-- PART 4: TOOL DEPENDENCIES
-- ============================================================================

-- 4.1 Refund Tool Dependencies
-- Refund REQUIRES Payment tool (hard dependency)
INSERT INTO tool_dependencies (
    tool_id,
    depends_on_tool_id,
    dependency_type,
    dependency_scope,
    auto_install,
    allow_disable,
    priority,
    install_order,
    description
)
SELECT
    t1.id,
    t2.id,
    'required',
    'runtime',
    TRUE,
    FALSE, -- Cannot disable Payment if Refund is active
    10,
    1, -- Payment must be installed first
    'Refund tool requires Payment tool to process refunds'
FROM tools t1
CROSS JOIN tools t2
WHERE t1.code = 'REFUND'
  AND t2.code = 'PAYMENT'
ON CONFLICT DO NOTHING;

-- 4.2 Payment Tool Dependencies
-- Payment recommends Webhook tool (for payment notifications)
INSERT INTO tool_dependencies (
    tool_id,
    depends_on_tool_id,
    dependency_type,
    dependency_scope,
    auto_install,
    allow_disable,
    priority,
    install_order,
    description
)
SELECT
    t1.id,
    t2.id,
    'recommended',
    'runtime',
    TRUE,
    TRUE, -- Can disable Webhook independently
    5,
    2, -- Webhook installed after Payment
    'Webhook tool recommended for real-time payment event notifications'
FROM tools t1
CROSS JOIN tools t2
WHERE t1.code = 'PAYMENT'
  AND t2.code = 'WEBHOOK'
ON CONFLICT DO NOTHING;

-- ============================================================================
-- Success message
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Registry seed data created successfully!';
    RAISE NOTICE '  - 6 modules inserted';
    RAISE NOTICE '  - 3 tools inserted';
    RAISE NOTICE '  - 4 module dependencies defined';
    RAISE NOTICE '  - 2 tool dependencies defined';
END $$;
