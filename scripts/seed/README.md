# 🌱 Database Seeding Guide

## 🎓 Seed Nedir ve Ne İçin Kullanılır?

**Seed (Tohum) Verisi:** Development ve test ortamları için örnek veri oluşturma scripti.

### Kullanım Alanları:
- 🧪 **Testing:** Automated testler için fixture data
- 💻 **Development:** Local development için örnek data
- 🎯 **Demo:** Product demo için sample data
- 🏗️ **Staging:** Pre-production testing environment

### ⚠️ Production'da Kullanılmaz!
Seed scripti production database'de **ASLA** çalıştırılmamalıdır.

---

## 🛡️ Production Guard Mekanizması

### Nasıl Çalışır?

```sql
-- 1. Environment variable check
v_environment := current_setting('app.environment', true);

IF v_environment = 'production' THEN
    RAISE EXCEPTION 'Cannot seed production'
END IF;

-- 2. Database name check
IF current_database() LIKE '%prod%' THEN
    RAISE EXCEPTION 'Production database detected'
END IF;
```

### Guard Layers:
1. **PostgreSQL Config:** `app.environment` variable
2. **Database Name:** `*prod*` pattern check
3. **Manual Confirmation:** Makefile prompt (future)

---

## 🚀 Kullanım

### Development Environment

```bash
# PostgreSQL'de environment set et
psql -d nexspaces_dev -c "ALTER DATABASE nexspaces_dev SET app.environment = 'development';"

# Seed script'i çalıştır
psql -d nexspaces_dev -f scripts/seed/dev_seed.sql

# Çıktı:
# NOTICE:  SEED GUARD: Environment check passed (environment: development, database: nexspaces_dev)
# ... seed işlemleri ...
```

### Staging Environment

```bash
psql -d nexspaces_staging -c "ALTER DATABASE nexspaces_staging SET app.environment = 'staging';"
psql -d nexspaces_staging -f scripts/seed/dev_seed.sql
```

### ❌ Production'da (Engellenecek)

```bash
psql -d nexspaces_production -f scripts/seed/dev_seed.sql

# Çıktı:
# ERROR:  SEED GUARD: Development seed script cannot run in production environment.
```

---

## 📁 Seed Script Yapısı

```
scripts/seed/
├── README.md              # Bu dosya
├── dev_seed.sql           # Development seed data
├── test_seed.sql          # Test fixture data (future)
└── demo_seed.sql          # Demo/showcase data (future)
```

---

## 🎓 Backend Best Practices

### Seed vs Migration

| Aspect | Migration | Seed |
|--------|-----------|------|
| Purpose | Schema changes | Sample data |
| Environment | ALL (dev, staging, prod) | ONLY dev/staging |
| Idempotency | Required | Optional |
| Rollback | Must have down script | Not needed |
| Production | ✅ Safe | ❌ Dangerous |

### Seed Data Principles

1. **Idempotent:** Multiple runs should be safe
   ```sql
   ON CONFLICT (email, tenant_id) DO NOTHING
   ```

2. **Deterministic IDs:** Use fixed UUIDs for testing
   ```sql
   v_tenant_id UUID := '550e8400-e29b-41d4-a716-446655440000'::UUID
   ```

3. **Clear Documentation:** Comment what each section does

4. **Environment-Aware:** Check environment before executing

---

## 🔧 CI/CD Integration

### GitHub Actions Example

```yaml
name: Run Tests with Seed Data

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: nexspaces_test
          POSTGRES_PASSWORD: password

    steps:
      - name: Set test environment
        run: |
          psql -c "ALTER DATABASE nexspaces_test SET app.environment = 'test';"

      - name: Run migrations
        run: make migrate-up

      - name: Seed test data
        run: psql -f scripts/seed/dev_seed.sql

      - name: Run integration tests
        run: go test -v ./tests/integration/...
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  db:
    image: postgres:15
    environment:
      POSTGRES_DB: nexspaces_dev
      POSTGRES_PASSWORD: password
    command: postgres -c app.environment=development
    volumes:
      - ./scripts/seed:/docker-entrypoint-initdb.d/seed
```

---

## 🎯 Future Improvements

- [ ] **Makefile target:** `make seed-dev`, `make seed-test`
- [ ] **Go seed runner:** Programmatic seeding from Go code
- [ ] **Seed versioning:** Track which seed version is applied
- [ ] **Seed templates:** Parametrize seed data (tenant count, user count)
- [ ] **Reset script:** Clean database and re-seed
- [ ] **Faker integration:** Generate realistic random data

---

## 📚 References

- [PostgreSQL Custom Variables](https://www.postgresql.org/docs/current/sql-set.html)
- [Database Seeding Best Practices](https://12factor.net/dev-prod-parity)
- [Test Fixtures vs Seeds](https://en.wikipedia.org/wiki/Test_fixture)
