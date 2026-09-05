# Keystone Platform - Modül ve Araç Envanteri

**Oluşturulma Tarihi:** 2025-10-05
**Week 1 Day 1:** Mevcut Domain Analizi ve Kategorileştirme

---

## 📋 Executive Summary

- **Toplam Domain:** 13
- **Modüller:** 6
- **Global Araçlar:** 3
- **Platform Infrastructure:** 4

---

## 🏗️ 1. MODÜLLER (Vertical-Specific Features)

Modüller, belirli sektörlere/kullanım senaryolarına özel işlevsellik sağlar. Her modül tenant tarafından aktif edilebilir/deaktif edilebilir.

### 1.1 LMS (Learning Management System)
**Domain:** `internal/domain/lesson/`

**Entities:**
- `Lesson` - Ana ders/oturum entity
- `Assignment` - Ödev yönetimi
- `Student` - Öğrenci bilgileri

**Özellikler:**
- One-on-one ve grup dersleri
- Online/In-person ders tipleri
- Homework tracking
- Performance scoring
- Attendance management
- Material/resource yönetimi
- Meeting URL entegrasyonu (Zoom/Meet)

**Status:** ✅ Implemented
**Kategori:** Modül
**Target Sektör:** Education, Tutoring, Training

---

### 1.2 CMS/Blog Module
**Domains:** `internal/domain/blog/`, `internal/domain/website/`

**Entities:**
- `Post` - Blog yazıları
- `Category` - Kategori yönetimi
- `Website` - Multi-tenant website entity

**Özellikler:**
- Draft/Published/Archived status
- Featured posts
- View count tracking
- Tag system
- Category organization
- Rich text content
- Custom domain support
- Template system integration

**Website Types (Built-in):**
- CMS
- CRM
- E-commerce
- Education
- Hospitality
- ERP
- Blog
- Portfolio
- Custom

**Status:** ✅ Implemented
**Kategori:** Modül
**Target Sektör:** Content creators, Bloggers, Publishers

---

### 1.3 Project Management Module
**Domain:** `internal/domain/project/`

**Entities:**
- `Project` - Portfolio project entity

**Özellikler:**
- Project status (Planned, In-Progress, Completed, Cancelled)
- Categories (Web Design, Mobile App, Automation, E-commerce, CMS, API)
- Client tracking
- Technology stack
- Image gallery (cover + multiple images)
- Live URL + GitHub URL
- Start/End date tracking
- Featured projects
- View count
- Sort order

**Status:** ✅ Implemented
**Kategori:** Modül
**Target Sektör:** Freelancers, Agencies, Developers

---

### 1.4 Booking/Appointment Module
**Domain:** `internal/domain/booking/`

**Entities:**
- `Appointment` - Randevu/rezervasyon entity
- `Availability` - Müsaitlik yönetimi

**Özellikler:**
- Appointment types (Consultation, Meeting, Lesson, Interview)
- Status management (Pending, Confirmed, Cancelled, Completed, No-show)
- Client information tracking
- Time slot management
- Meeting URL support
- Reminder system
- Cancellation management

**Status:** ✅ Implemented
**Kategori:** Modül
**Target Sektör:** Healthcare, Consulting, Service providers, HMS

**Not:** Bu modül hem genel booking hem de HMS (Hotel Management System) için kullanılabilir.

---

### 1.5 Service/Product Catalog Module
**Domain:** `internal/domain/service/`

**Entities:**
- `Service` - Hizmet/Ürün entity

**Özellikler:**
- Service categories (Consulting, Training, Support, Other)
- Pricing models:
  - One-time
  - Recurring
  - Subscription
  - Usage-based
- Billing cycle (monthly, yearly)
- Duration tracking
- Max clients limit
- Public/Private visibility
- Featured services
- Image support

**Status:** ✅ Implemented
**Kategori:** Modül
**Target Sektör:** Service providers, SaaS, Consultants

---

### 1.6 Multi-Tenant Website Builder (Platform Core)
**Domain:** `internal/domain/website/`

**Özellik:** Her tenant kendi website'ını oluşturur ve modüllerini seçer.

**Built-in Website Types:**
- CMS → Blog module
- CRM → User/Service modules
- E-commerce → Service/Payment modules
- Education (LMS) → Lesson module
- Hospitality (HMS) → Booking module
- ERP → Project/Service modules
- Blog → Blog module
- Portfolio → Project module

**Status:** ✅ Implemented
**Kategori:** Platform Infrastructure (Module Selector)

---

## 🔧 2. GLOBAL ARAÇLAR (Cross-Module Tools)

Global araçlar tüm modüller tarafından kullanılabilir ve tenant bazlı aktif edilir.

### 2.1 Payment Tool
**Domain:** `internal/domain/payment/`

**Entities:**
- `Payment` - Ödeme transaction entity

**Payment Providers:**
- ✅ Iyzico (Turkey)
- ✅ Checkout.com
- ✅ Stripe

**Özellikler:**
- Multi-provider support
- 3DS authentication (3D Secure 2.0)
- Installment support (Turkey-specific)
- Card information (PCI-compliant, no full PAN)
- Customer information
- Billing address
- Payment status (Pending, Processing, Requires3DS, Succeeded, Failed, Canceled, Refunded)
- Provider webhook integration
- Metadata support
- Idempotency

**Status:** ✅ Implemented
**Kategori:** Global Araç
**Dependencies:** Webhook araç

---

### 2.2 Refund Tool
**Domain:** `internal/domain/refund/`

**Entities:**
- `Refund` - İade transaction entity

**Özellikler:**
- Multi-provider support (Iyzico, Checkout, Stripe)
- Refund status tracking
- Refund reasons (Customer request, Duplicate, Fraudulent, Other)
- Full/partial refund support
- Provider integration
- Audit trail

**Status:** ✅ Implemented
**Kategori:** Global Araç
**Dependencies:** Payment araç (required)

---

### 2.3 Webhook Tool
**Domain:** `internal/domain/webhook/`

**Entities:**
- `PaymentEvent` - Webhook event entity

**Event Types:**
- Payment events (succeeded, failed, pending, canceled)
- Refund events (succeeded, failed, pending)
- 3DS events (required, succeeded, failed)
- Chargeback events (created, updated)

**Özellikler:**
- Provider-agnostic event handling
- Idempotency (provider_event_id)
- Signature validation
- Retry mechanism
- Processing status tracking
- Request metadata (IP, headers)
- Event correlation (payment_id, refund_id)

**Status:** ✅ Implemented
**Kategori:** Global Araç
**Dependencies:** Payment/Refund araçları ile entegre

---

## 🏛️ 3. PLATFORM INFRASTRUCTURE (Core System)

Platform'un temel altyapısı - her tenant için zorunlu.

### 3.1 Tenant Management
**Domain:** `internal/domain/tenant/`

**Entities:**
- `Tenant` - Multi-tenant isolation entity

**Özellikler:**
- Tenant isolation (Row Level Security)
- Subdomain support (tenant.keystone.dev)
- Custom domain support
- Subscription management
- Feature flags per tenant
- Tenant metadata

**Status:** ✅ Implemented
**Kategori:** Platform Infrastructure

---

### 3.2 User Management
**Domain:** `internal/domain/user/`

**Entities:**
- `User` - User entity with tenant isolation

**Özellikler:**
- Multi-tenant user management
- Role-based access control (RBAC)
- User authentication (Better Auth planned)
- User permissions
- User metadata

**Status:** ✅ Implemented
**Kategori:** Platform Infrastructure

---

### 3.3 Template System
**Domain:** `internal/domain/template/` (referenced in website entity)

**Status:** ⚠️ Not yet implemented (directory not found)
**Planned:** Template marketplace, template versioning
**Kategori:** Platform Infrastructure

---

## 📊 4. MODULE-TOOL MAPPING

Bu bölüm hangi modülün hangi araçları kullandığını gösterir.

### LMS (Lesson Module)
**Default Tools:**
- User Management (required)

**Optional Tools:**
- Payment (ücretli ders sistemleri için)
- Booking (ders randevuları için - overlap with lesson scheduling)

---

### CMS/Blog Module
**Default Tools:**
- User Management (required)

**Optional Tools:**
- None (standalone content module)

---

### Project Management Module
**Default Tools:**
- User Management (required)

**Optional Tools:**
- None (standalone portfolio module)

---

### Booking/Appointment Module
**Default Tools:**
- User Management (required)

**Optional Tools:**
- Payment (randevu ödemesi için)
- Webhook (payment notifications)

---

### Service Catalog Module
**Default Tools:**
- User Management (required)

**Optional Tools:**
- Payment (servis satın alma için)
- Refund (iade işlemleri için)
- Webhook (payment/refund notifications)

---

## 🔗 5. DEPENDENCIES & RELATIONSHIPS

### Dependency Graph

```
Platform Infrastructure (Always Active)
├── Tenant Management ────────┐
├── User Management ──────────┤
└── Template System (planned) │
                               │
Modules (Tenant Selects)       │
├── LMS ──────────────────────┤
├── CMS/Blog ─────────────────┤
├── Project Management ───────┤
├── Booking ──────────────────┤
├── Service Catalog ──────────┤
└── Website Builder ──────────┘
         │
         └──> Global Tools (Optional)
              ├── Payment ────> Webhook
              ├── Refund ─────> Payment (dependency)
              └── Webhook (standalone)
```

### Hard Dependencies
- **Refund** → **Payment** (cannot exist without payment)
- **Webhook** → **Payment/Refund** (event-driven architecture)
- All Modules → **Tenant** (multi-tenant isolation)
- All Modules → **User** (access control)

### Soft Dependencies (Optional)
- **LMS** ↔ **Payment** (paid lessons)
- **Booking** ↔ **Payment** (paid appointments)
- **Service** ↔ **Payment** (service purchases)
- **Service** ↔ **Refund** (service refunds)

---

## 🎯 6. MARKETPLACE CATEGORIZATION

Bu bölüm kullanıcının marketplace'te nasıl bir deneyim yaşayacağını gösterir.

### Use Case 1: Education Platform (LMS)
**Tenant selects:**
- ✅ LMS Module (default)
- ✅ Payment Tool (optional - for paid courses)
- ✅ Webhook Tool (optional - payment notifications)

**Auto-included:**
- Tenant Management
- User Management

---

### Use Case 2: Hotel Management (HMS)
**Tenant selects:**
- ✅ Booking Module (default - for room reservations)
- ✅ Service Module (optional - for hotel services)
- ✅ Payment Tool (optional - for bookings/services)
- ✅ Webhook Tool (optional - payment notifications)

**Auto-included:**
- Tenant Management
- User Management

---

### Use Case 3: E-commerce Platform
**Tenant selects:**
- ✅ Service Module (as product catalog)
- ✅ Payment Tool (required - for purchases)
- ✅ Refund Tool (optional - for returns)
- ✅ Webhook Tool (required - order notifications)

**Auto-included:**
- Tenant Management
- User Management

---

### Use Case 4: Freelancer Portfolio + Booking
**Tenant selects:**
- ✅ Project Module (portfolio showcase)
- ✅ Booking Module (client appointments)
- ✅ Blog Module (thought leadership)
- ✅ Payment Tool (optional - booking deposits)

**Auto-included:**
- Tenant Management
- User Management

---

## 📝 7. MISSING MODULES & TOOLS (For Roadmap)

### 🚧 Modules to Build

1. **Inventory Module** (for E-commerce)
   - Product inventory
   - Stock management
   - Variant support

2. **CRM Module**
   - Customer management
   - Lead tracking
   - Sales pipeline

3. **Quiz/Assessment Engine** (LMS-specific tool)
   - Quiz creation
   - Auto-grading
   - Progress tracking

4. **Room Management** (HMS-specific tool)
   - Room types
   - Availability calendar
   - Pricing tiers

---

### 🚧 Global Tools to Build

1. **Chat Tool** (WebSocket)
   - Real-time messaging
   - Tenant isolation
   - Chat rooms/channels
   - File sharing

2. **File Upload Tool**
   - Multi-tenant storage (S3)
   - Image optimization
   - File type validation
   - Quota management

3. **Notification Tool**
   - Email notifications
   - SMS notifications
   - Push notifications
   - Template management

4. **N8n Automation Tool**
   - Workflow automation
   - Integration hub
   - Event-driven actions

5. **Analytics Tool**
   - Usage metrics
   - Revenue tracking
   - User behavior
   - Custom reports

---

## 🎨 8. TEMPLATE SYSTEM (Planned)

**Status:** Not yet implemented

**Purpose:**
Templates combine modules + tools into ready-to-use solutions.

### Example Templates

#### 1. "School LMS" Template
**Includes:**
- LMS Module
- Payment Tool (for course fees)
- Webhook Tool
- Pre-configured lesson types
- Sample curriculum

#### 2. "Boutique Hotel" Template
**Includes:**
- Booking Module (room reservations)
- Service Module (room service, spa)
- Payment Tool
- Webhook Tool
- Pre-configured room types

#### 3. "Freelancer Portfolio" Template
**Includes:**
- Project Module
- Blog Module
- Booking Module (client meetings)
- Pre-designed portfolio layouts

---

## ✅ 9. NEXT STEPS (Week 1)

### Day 2-3: Database Schema
- [ ] Create `modules` table (catalog)
- [ ] Create `tools` table (catalog)
- [ ] Create `tenant_modules` table (active modules per tenant)
- [ ] Create `tenant_tools` table (active tools per tenant)
- [ ] Define module-tool dependencies table

### Day 4: Seed Data
- [ ] Populate `modules` table with 6 existing modules
- [ ] Populate `tools` table with 3 existing tools
- [ ] Create sample tenant with LMS + Payment active
- [ ] Add module/tool metadata (pricing, descriptions, features)

### Day 5: Repository & Service Layer
- [ ] Create `ModuleRepository` (CRUD)
- [ ] Create `ToolRepository` (CRUD)
- [ ] Create `TenantModuleService` (activation logic)
- [ ] Create `DependencyCheckerService` (validate dependencies)

---

## 📌 10. NOTES & OBSERVATIONS

### ✅ Strengths
1. **Clean domain separation** - Each domain is well-isolated
2. **Multi-tenant awareness** - Every entity has `tenant_id`
3. **Rich entity models** - Entities have comprehensive business logic
4. **Status management** - Clear state machines for entities
5. **Audit trails** - CreatedAt/UpdatedAt/CreatedBy/UpdatedBy patterns

### ⚠️ Areas for Improvement
1. **Template system missing** - Critical for marketplace
2. **Module-tool relationships** - Not explicitly defined in code yet
3. **Dependency injection** - Some entities hardcode provider names
4. **Cross-module data sharing** - Need event bus for decoupling

### 🔐 Security Observations
1. **Tenant isolation** - ✅ All entities have tenant_id
2. **PII handling** - ✅ Payment domain doesn't store full PAN
3. **Webhook security** - ✅ Signature validation implemented
4. **Audit logging** - ✅ Comprehensive audit fields

---

**End of Inventory Report**
**Generated by:** Claude Code
**Date:** 2025-10-05
