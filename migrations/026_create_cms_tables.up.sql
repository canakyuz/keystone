-- ============================================================================
-- Migration: 026 - CMS Page Builder Tables
-- Description: Pages, Sections, Components için multi-tenant CMS sistemi
-- Created: 2025-10-13
-- ============================================================================

-- ============================================================================
-- 1. PAGES TABLE (Ana Sayfalar)
-- ============================================================================
-- Her tenant için page listesi (homepage, about, contact, vs.)
CREATE TABLE IF NOT EXISTS pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(255) NOT NULL,                    -- URL slug: "/", "/about", "/blog"

    -- Multi-language support (JSONB)
    title JSONB NOT NULL,                          -- {"tr": "Anasayfa", "en": "Home"}
    meta_description JSONB,                        -- SEO için meta description
    meta_keywords JSONB,                           -- SEO keywords

    -- Page settings
    template_type VARCHAR(50) DEFAULT 'custom',    -- "landing", "blog-list", "custom"
    is_published BOOLEAN DEFAULT false,            -- Yayında mı?
    is_homepage BOOLEAN DEFAULT false,             -- Ana sayfa mı?

    -- Publishing
    published_at TIMESTAMP,

    -- Audit fields
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,

    -- Constraints
    CONSTRAINT pages_slug_check CHECK (slug ~ '^/[a-z0-9-]*$' OR slug = '/')
);

-- Indexes
CREATE INDEX idx_pages_slug ON pages(slug);
CREATE INDEX idx_pages_published ON pages(is_published, published_at);
CREATE INDEX idx_pages_created ON pages(created_at DESC);

-- ============================================================================
-- 2. SECTIONS TABLE (Sayfadaki Bölümler)
-- ============================================================================
-- Her page içinde sıralı section'lar (hero, features, cta, vs.)
CREATE TABLE IF NOT EXISTS sections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id UUID NOT NULL REFERENCES pages(id) ON DELETE CASCADE,

    -- Section type (hero, features, testimonials, cta, custom)
    section_type VARCHAR(50) NOT NULL,

    -- Sıralama (0'dan başlar)
    order_index INT NOT NULL DEFAULT 0,

    -- Section configuration (JSONB)
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    -- Example config:
    -- {
    --   "background_color": "#ffffff",
    --   "background_image": "https://...",
    --   "padding": "large",           // small, medium, large
    --   "full_width": true,
    --   "container_width": "1200px",
    --   "custom_css": "margin-top: 20px;"
    -- }

    -- Visibility
    is_visible BOOLEAN DEFAULT true,

    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT sections_order_positive CHECK (order_index >= 0)
);

-- Indexes
CREATE INDEX idx_sections_page ON sections(page_id, order_index);
CREATE INDEX idx_sections_type ON sections(section_type);
CREATE INDEX idx_sections_visible ON sections(is_visible);

-- ============================================================================
-- 3. COMPONENTS TABLE (Section İçindeki Micro Components)
-- ============================================================================
-- Her section içinde component'ler (button, text, image, form, vs.)
CREATE TABLE IF NOT EXISTS components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    section_id UUID NOT NULL REFERENCES sections(id) ON DELETE CASCADE,

    -- Component type (button, text, heading, image, video, form, etc.)
    component_type VARCHAR(50) NOT NULL,

    -- Sıralama (0'dan başlar)
    order_index INT NOT NULL DEFAULT 0,

    -- Component properties (JSONB)
    props JSONB NOT NULL DEFAULT '{}'::jsonb,
    -- Example props for button:
    -- {
    --   "text": {"tr": "İletişime Geç", "en": "Contact Us"},
    --   "href": "/contact",
    --   "variant": "primary",        // primary, secondary, outline
    --   "size": "large",              // small, medium, large
    --   "icon": "arrow-right",
    --   "open_new_tab": false
    -- }
    --
    -- Example props for text:
    -- {
    --   "content": {"tr": "Hoş geldiniz", "en": "Welcome"},
    --   "variant": "h1",              // h1, h2, h3, p, span
    --   "align": "center",            // left, center, right
    --   "color": "#333333",
    --   "font_size": "48px",
    --   "font_weight": "bold"
    -- }
    --
    -- Example props for image:
    -- {
    --   "src": "https://cdn.nexpaces.com/images/hero.jpg",
    --   "alt": {"tr": "Hero resmi", "en": "Hero image"},
    --   "width": "100%",
    --   "height": "auto",
    --   "object_fit": "cover"
    -- }

    -- Visibility
    is_visible BOOLEAN DEFAULT true,

    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT components_order_positive CHECK (order_index >= 0)
);

-- Indexes
CREATE INDEX idx_components_section ON components(section_id, order_index);
CREATE INDEX idx_components_type ON components(component_type);
CREATE INDEX idx_components_visible ON components(is_visible);

-- ============================================================================
-- 4. LANGUAGES TABLE (Aktif Diller)
-- ============================================================================
-- Tenant'ın aktif olarak kullandığı diller
CREATE TABLE IF NOT EXISTS languages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(5) NOT NULL UNIQUE,               -- "tr", "en", "de", "fr"
    name VARCHAR(50) NOT NULL,                     -- "Türkçe", "English", "Deutsch"
    native_name VARCHAR(50) NOT NULL,              -- "Türkçe", "English", "Deutsch"
    is_default BOOLEAN DEFAULT false,              -- Varsayılan dil mi?
    is_active BOOLEAN DEFAULT true,                -- Aktif mi?
    flag_emoji VARCHAR(10),                        -- "🇹🇷", "🇬🇧", "🇩🇪"

    -- Display order
    order_index INT DEFAULT 0,

    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_languages_active ON languages(is_active, order_index);
CREATE INDEX idx_languages_default ON languages(is_default);

-- ============================================================================
-- 5. SEED DEFAULT LANGUAGES
-- ============================================================================
INSERT INTO languages (code, name, native_name, is_default, is_active, flag_emoji, order_index) VALUES
('tr', 'Turkish', 'Türkçe', true, true, '🇹🇷', 0),
('en', 'English', 'English', false, true, '🇬🇧', 1),
('de', 'German', 'Deutsch', false, false, '🇩🇪', 2),
('fr', 'French', 'Français', false, false, '🇫🇷', 3)
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- 6. HELPER FUNCTIONS
-- ============================================================================

-- Function: Get translated text from JSONB based on language
CREATE OR REPLACE FUNCTION get_translation(content JSONB, lang_code VARCHAR(5))
RETURNS TEXT AS $$
BEGIN
    -- Try requested language
    IF content ? lang_code THEN
        RETURN content ->> lang_code;
    END IF;

    -- Fallback to default language (tr)
    IF content ? 'tr' THEN
        RETURN content ->> 'tr';
    END IF;

    -- Fallback to first available language
    RETURN (SELECT value FROM jsonb_each_text(content) LIMIT 1);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function: Reorder sections after insert/delete
CREATE OR REPLACE FUNCTION reorder_sections()
RETURNS TRIGGER AS $$
BEGIN
    -- After delete, reorder remaining sections
    IF TG_OP = 'DELETE' THEN
        UPDATE sections
        SET order_index = subquery.new_index
        FROM (
            SELECT id, ROW_NUMBER() OVER (ORDER BY order_index) - 1 AS new_index
            FROM sections
            WHERE page_id = OLD.page_id
        ) AS subquery
        WHERE sections.id = subquery.id;

        RETURN OLD;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function: Reorder components after insert/delete
CREATE OR REPLACE FUNCTION reorder_components()
RETURNS TRIGGER AS $$
BEGIN
    -- After delete, reorder remaining components
    IF TG_OP = 'DELETE' THEN
        UPDATE components
        SET order_index = subquery.new_index
        FROM (
            SELECT id, ROW_NUMBER() OVER (ORDER BY order_index) - 1 AS new_index
            FROM components
            WHERE section_id = OLD.section_id
        ) AS subquery
        WHERE components.id = subquery.id;

        RETURN OLD;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 7. TRIGGERS
-- ============================================================================

-- Trigger: Auto-reorder sections
CREATE TRIGGER trigger_reorder_sections
AFTER DELETE ON sections
FOR EACH ROW
EXECUTE FUNCTION reorder_sections();

-- Trigger: Auto-reorder components
CREATE TRIGGER trigger_reorder_components
AFTER DELETE ON components
FOR EACH ROW
EXECUTE FUNCTION reorder_components();

-- Trigger: Update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_pages_updated_at
BEFORE UPDATE ON pages
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_sections_updated_at
BEFORE UPDATE ON sections
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_components_updated_at
BEFORE UPDATE ON components
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();

-- ============================================================================
-- 8. COMMENTS (Documentation)
-- ============================================================================

COMMENT ON TABLE pages IS 'CMS sayfaları - multi-language destekli';
COMMENT ON COLUMN pages.title IS 'JSONB format: {"tr": "Başlık", "en": "Title"}';
COMMENT ON COLUMN pages.slug IS 'URL slug, "/" ile başlamalı';
COMMENT ON COLUMN pages.is_homepage IS 'Ana sayfa flagı (tenant başına 1 tane olmalı)';

COMMENT ON TABLE sections IS 'Page içindeki bölümler (hero, features, cta, vs.)';
COMMENT ON COLUMN sections.config IS 'Section özellikleri (background, padding, vs.)';
COMMENT ON COLUMN sections.order_index IS 'Sıralama (0-indexed, auto-reorder on delete)';

COMMENT ON TABLE components IS 'Section içindeki micro component\'ler (button, text, image, vs.)';
COMMENT ON COLUMN components.props IS 'Component özellikleri (variant, size, text, vs.)';
COMMENT ON COLUMN components.order_index IS 'Sıralama (0-indexed, auto-reorder on delete)';

COMMENT ON TABLE languages IS 'Aktif diller ve metadata';

-- ============================================================================
-- Migration Complete ✅
-- ============================================================================
