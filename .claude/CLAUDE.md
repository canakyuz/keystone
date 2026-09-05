# Keystone API - Claude Code Kuralları

> **Multi-Tenant SaaS Backend (Go + Fiber + PostgreSQL)**
> **Mimari:** Clean Architecture + Schema-per-Tenant

---

## ⚠️ YASAKLAR (Zaman Kaybı)

- **❌ Mock/Test Data:** Gerçek PostgreSQL kullan
- **❌ Unit Test Obsesyonu:** Sadece kritik business logic test et
- **❌ İki Aşamalı Yaklaşım:** Direkt implementation yap
- **❌ Test-First:** Önce kodu yaz, sonra test ekle
- **✅ ZORUNLU:** Her anlamlı değişiklikten sonra git commit at

---

## 🎯 Öncelikler

1. **Tenant Isolation** - Tenant'lar arası veri sızıntısı = 0
2. **Clean Architecture** - Katman ayrımı mutlak
3. **Production Ready** - Her commit production'a gidebilir
4. **Performance** - P95 <200ms
5. **Security** - SOC 2 + GDPR uyumlu

---

## 🏗️ Mimari Katmanlar

```
Handlers (HTTP)
    ↓
Services (Business Logic)
    ↓
Entities (Domain)
    ↓
Repository (Database)
```

**Kural:** İçteki katman dıştaki katmana bağımlı OLAMAZ.

---

## 🔒 Multi-Tenant Kuralları

### Her Request'te Zorunlu Context
```go
type RequestContext struct {
    TenantID   string `validate:"required"`
    SchemaName string `validate:"required"`
    UserID     string `validate:"required"`
}
```

### Database İzolasyonu
```go
// ❌ YANLIŞ: Tenant context yok
db.Query("SELECT * FROM users WHERE email = $1", email)

// ✅ DOĞRU: Schema scoping
db.Exec("SET search_path TO tenant_acme, public")
db.Query("SELECT * FROM users WHERE email = $1", email)
```

### Cache İzolasyonu
```typescript
// ✅ Tenant prefix zorunlu
const cacheKey = `tenant:${tenantId}:${resource}:${id}`;
```

---

## 💎 Kod Kuralları

### DRY
```go
// ≥3 tekrar görürsen, fonksiyon çıkar
```

### KISS
```go
// En basit çalışan çözüm. Gereksiz soyutlama yapma.
```

### YAGNI
```go
// Bugün gerekmeyen özelliği ekleme.
```

### Single Responsibility
```go
// Her servis/fonksiyon tek iş yapsın
```

---

## 🔐 Güvenlik

### Input Validation
```go
type CreateTenantRequest struct {
    Name  string `validate:"required,min=3,max=100"`
    Email string `validate:"required,email"`
}
```

### SQL Injection
```go
// ❌ String concatenation
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)

// ✅ Parameterized query
db.Query("SELECT * FROM users WHERE email = $1", email)
```

### Error Messages
```go
// ❌ Implementation detail leak
return fmt.Errorf("database error: %v", err)

// ✅ Generic message, internal logging
logger.Error("db_error", zap.Error(err))
return errors.New("failed to process request")
```

---

## 📊 Performance

### N+1 Engellemek
```go
// ❌ Loop içinde query
for _, user := range users {
    roles := repo.GetUserRoles(user.ID) // BAD!
}

// ✅ Batch query
userIDs := extractIDs(users)
rolesByUserID := repo.GetRolesByUserIDs(userIDs) // GOOD!
```

### Index Stratejisi
```sql
-- ❌ Tenant_id yok
CREATE INDEX idx_users_email ON users(email);

-- ✅ Tenant_id ilk sırada
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);
```

### Cache Stratejisi
```go
// L1: App cache (5 min)
// L2: Redis (15 min)
// L3: Database
// Key format: "tenant:{tenant_id}:{resource}:{id}"
```

---

## 🚀 Git Commit Standartları

```bash
# Conventional commits
feat(tenant): add schema provisioning
fix(auth): prevent cross-tenant session leak
refactor(user): extract validation logic
perf(db): optimize tenant queries
```

### Commit Öncesi Kontrol
- [ ] Testler geçiyor mu?
- [ ] Lint temiz mi?
- [ ] Build başarılı mı?
- [ ] Hassas veri yok mu?
- [ ] Tenant isolation korunuyor mu?

---

## 📝 Dokümantasyon

### Yorumlar
```go
// ❌ NE yaptığını açıklama
counter++ // Increment counter

// ✅ NEDEN yaptığını açıkla
// Exponential backoff kullanıyoruz çünkü payment API'si
// yüksek trafikte rate limit yapıyor. Max 3 retry, 8s delay.
```

---

## 🚨 Anti-Patterns (ASLA YAPMA)

```go
// ❌ Hardcoded credentials
const dbPassword = "secret"

// ❌ Library'de panic
func ProcessData(data string) {
    if data == "" {
        panic("empty data") // WRONG
    }
}

// ❌ Error ignore
db.Exec("UPDATE users SET active = true")

// ❌ Global mutable state
var currentTenantID string // Race condition

// ❌ Handler'da database logic
func (h *Handler) GetUser(c *fiber.Ctx) error {
    row := h.db.QueryRow("SELECT ...") // WRONG
}
```

---

## 📦 Proje Yapısı

```
keystone/
├── cmd/server/              # Entry point
├── internal/
│   ├── domain/              # Entities (pure)
│   ├── usecase/             # Services (business logic)
│   ├── repository/          # Data access
│   └── handlers/            # HTTP handlers
├── pkg/                     # Shared utilities
├── migrations/              # Database migrations
└── scripts/seed/            # Dev seed data only
```

---

## ✅ Definition of Done

1. **Kod Kalitesi**
   - [ ] Clean architecture uyumlu
   - [ ] DRY, KISS, YAGNI uygulandı
   - [ ] Lint temiz

2. **Güvenlik**
   - [ ] Input validation var
   - [ ] Tenant isolation doğrulandı
   - [ ] Logda hassas veri yok
   - [ ] SQL injection korumalı

3. **Performance**
   - [ ] N+1 query yok
   - [ ] Index optimize
   - [ ] Cache stratejisi var

4. **Production Ready**
   - [ ] Error handling tam
   - [ ] Structured logging
   - [ ] Migration backward-compatible

---

**Son Güncelleme:** Ekim 2025
**Versiyon:** 2.0
