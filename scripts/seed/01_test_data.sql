-- Test Data Seed Script for NexSpaces API
-- This script creates test tenant and users for development/testing

-- ============================================
-- 1. TENANT: canakyuz.co
-- ============================================
INSERT INTO tenants (
    id,
    name,
    slug,
    domain,
    status,
    plan,
    max_users,
    settings,
    created_at,
    updated_at
) VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Can Akyüz Personal Site',
    'canakyuz',
    'canakyuz.co',
    'active',
    'professional',
    100,
    '{"theme": "dark", "language": "tr", "timezone": "Europe/Istanbul"}'::jsonb,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 2. USERS
-- ============================================

-- Admin user (password: Admin123!)
-- Bcrypt hash for: Admin123!
INSERT INTO users (
    id,
    tenant_id,
    email,
    password_hash,
    first_name,
    last_name,
    role,
    status,
    email_verified,
    created_at,
    updated_at
) VALUES (
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'admin@canakyuz.co',
    '$2a$10$X7KV4nqQ7L5YZ8Y5Z8Y5ZeO5Z8Y5Z8Y5Z8Y5Z8Y5Z8Y5Z8Y5Z8Y5Z',
    'Can',
    'Akyüz',
    'admin',
    'active',
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Regular user (password: User123!)
INSERT INTO users (
    id,
    tenant_id,
    email,
    password_hash,
    first_name,
    last_name,
    role,
    status,
    email_verified,
    created_at,
    updated_at
) VALUES (
    'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'user@canakyuz.co',
    '$2a$10$Y8KV4nqQ7L5YZ8Y5Z8Y5ZeO5Z8Y5Z8Y5Z8Y5Z8Y5Z8Y5Z8Y5Z8Y5Y',
    'Test',
    'User',
    'user',
    'active',
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 3. BLOG CATEGORIES
-- ============================================
INSERT INTO blog_categories (
    id,
    tenant_id,
    name,
    slug,
    description,
    post_count,
    metadata,
    created_at,
    updated_at
) VALUES
(
    'd3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Teknoloji',
    'teknoloji',
    'Teknoloji ve yazılım geliştirme hakkında yazılar',
    0,
    '{"color": "#3B82F6"}'::jsonb,
    NOW(),
    NOW()
),
(
    'e4eebc99-9c0b-4ef8-bb6d-6bb9bd380a55',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Kişisel Gelişim',
    'kisisel-gelisim',
    'Kişisel gelişim ve kariyer üzerine yazılar',
    0,
    '{"color": "#10B981"}'::jsonb,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 4. BLOG POSTS
-- ============================================
INSERT INTO blog_posts (
    id,
    tenant_id,
    category_id,
    title,
    slug,
    content,
    excerpt,
    status,
    featured,
    view_count,
    image,
    tags,
    published_at,
    metadata,
    created_at,
    updated_at,
    created_by,
    updated_by
) VALUES (
    'f5eebc99-9c0b-4ef8-bb6d-6bb9bd380a66',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'd3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44',
    'NexSpaces: Multi-Tenant SaaS Platform',
    'nexspaces-multi-tenant-saas',
    '# NexSpaces Nedir?

NexSpaces, modern web uygulamaları için güçlü bir multi-tenant SaaS platformudur...

## Özellikler
- Multi-tenant architecture
- Role-based access control
- Template marketplace
- Scalable infrastructure',
    'NexSpaces multi-tenant SaaS platform hakkında detaylı bilgi',
    'published',
    true,
    42,
    '/images/blog/nexspaces-cover.jpg',
    ARRAY['saas', 'multi-tenant', 'go', 'nextjs'],
    NOW(),
    '{"readTime": 5, "author": "Can Akyüz"}'::jsonb,
    NOW(),
    NOW(),
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22'
) ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 5. SERVICES
-- ============================================
INSERT INTO services (
    id,
    tenant_id,
    name,
    slug,
    description,
    pricing_model,
    price,
    currency,
    duration,
    duration_unit,
    features,
    category,
    is_active,
    metadata,
    created_at,
    updated_at
) VALUES (
    '11eebc99-9c0b-4ef8-bb6d-6bb9bd380a77',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Web Development Consultation',
    'web-dev-consultation',
    'Professional web development consultation for your projects',
    'one_time',
    15000.00,
    'TRY',
    60,
    'minutes',
    ARRAY['Code review', 'Architecture design', 'Best practices', 'Q&A session'],
    'consulting',
    true,
    '{"bookingUrl": "https://cal.com/canakyuz"}'::jsonb,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 6. STUDENTS (for Lessons module)
-- ============================================
INSERT INTO students (
    id,
    tenant_id,
    first_name,
    last_name,
    email,
    phone,
    date_of_birth,
    grade_level,
    status,
    notes,
    metadata,
    created_at,
    updated_at
) VALUES (
    '22eebc99-9c0b-4ef8-bb6d-6bb9bd380a88',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Ali',
    'Yılmaz',
    'ali.yilmaz@example.com',
    '+905551234567',
    '2005-03-15',
    '10',
    'active',
    'Matematik ve fizik dersleri alıyor',
    '{"parentPhone": "+905559876543"}'::jsonb,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 7. WEBSITE (canakyuz.co)
-- ============================================
INSERT INTO websites (
    id,
    tenant_id,
    name,
    slug,
    domain,
    template,
    status,
    settings,
    metadata,
    created_at,
    updated_at
) VALUES (
    '33eebc99-9c0b-4ef8-bb6d-6bb9bd380a99',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Can Akyüz - Personal Website',
    'canakyuz-co',
    'canakyuz.co',
    'personal',
    'published',
    '{
        "theme": "dark",
        "primaryColor": "#3B82F6",
        "logo": "/logo.png",
        "navigation": [
            {"label": "Ana Sayfa", "url": "/"},
            {"label": "Blog", "url": "/blog"},
            {"label": "Hizmetler", "url": "/services"},
            {"label": "İletişim", "url": "/contact"}
        ]
    }'::jsonb,
    '{"analytics": "GA-XXXXXXXXX"}'::jsonb,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- ============================================
-- SUCCESS MESSAGE
-- ============================================
DO $$
BEGIN
    RAISE NOTICE '✓ Test data seeded successfully!';
    RAISE NOTICE '  - Tenant: canakyuz.co (id: a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11)';
    RAISE NOTICE '  - Admin: admin@canakyuz.co (password: Admin123!)';
    RAISE NOTICE '  - User: user@canakyuz.co (password: User123!)';
END $$;
