# 🎓 NexSpaces Backend - Kapsamlı Öğrenme ve Geliştirme Rehberi

> **Hedef:** Go backend geliştirmeyi öğrenmek ve NexSpaces projesini tamamlamak
> **Seviye:** Başlangıç/Orta
> **Tahmini Süre:** 4-6 hafta

---

## 📚 İçindekiler

1. [Go Programlama Dili Temelleri](#1-go-programlama-dili-temelleri)
2. [Backend Mimarisi Nedir?](#2-backend-mimarisi-nedir)
3. [Proje Klasör Yapısı Açıklaması](#3-proje-klasör-yapısı-açıklaması)
4. [Database (Veritabanı) Kullanımı](#4-database-veritabanı-kullanımı)
5. [Docker Kullanımı](#5-docker-kullanımı)
6. [Kalan Görevler ve Nasıl Yapılır](#6-kalan-görevler-ve-nasıl-yapılır)
7. [Hata Ayıklama (Debugging)](#7-hata-ayıklama-debugging)
8. [Test Yazma](#8-test-yazma)
9. [Deployment (Canlıya Alma)](#9-deployment-canlıya-alma)

---

## 1. Go Programlama Dili Temelleri

### 1.1. Go Nedir?

Go (Golang), Google tarafından geliştirilen, hızlı, basit ve güvenli bir programlama dilidir. Backend geliştirme için çok popülerdir.

**Neden Go?**
- ✅ Çok hızlı (compiled language)
- ✅ Basit syntax (Python kadar kolay)
- ✅ Concurrency desteği (aynı anda çok iş yapabilir)
- ✅ Güçlü standart kütüphane

### 1.2. Temel Go Syntax

#### Değişkenler ve Tipler

```go
package main

import "fmt"

func main() {
    // Değişken tanımlama
    var name string = "Can"
    age := 25  // Kısa syntax (tip otomatik anlaşılır)

    // Yazdırma
    fmt.Println("Merhaba,", name)
    fmt.Printf("Yaşın: %d\n", age)
}
```

**Temel Tipler:**
```go
string        // "Merhaba"
int           // 42
float64       // 3.14
bool          // true, false
[]string      // String dizisi: ["a", "b", "c"]
map[string]int // Map: {"age": 25}
```

#### Fonksiyonlar

```go
// Basit fonksiyon
func topla(a int, b int) int {
    return a + b
}

// Çoklu return (Go'nun özelliği!)
func bol(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("sıfıra bölme hatası")
    }
    return a / b, nil
}

// Kullanım
sonuc, err := bol(10, 2)
if err != nil {
    fmt.Println("Hata:", err)
    return
}
fmt.Println("Sonuç:", sonuc)
```

#### Struct (Yapı)

```go
// Struct tanımlama (Class gibi ama basit)
type Student struct {
    Name  string
    Age   int
    Email string
}

// Kullanım
student := Student{
    Name:  "Ahmet",
    Age:   20,
    Email: "ahmet@example.com",
}

fmt.Println(student.Name) // Ahmet
```

#### Method (Struct'a fonksiyon ekleme)

```go
// Student struct'ına method ekleme
func (s *Student) GetInfo() string {
    return fmt.Sprintf("%s (%d yaşında)", s.Name, s.Age)
}

// Kullanım
student := Student{Name: "Ahmet", Age: 20}
fmt.Println(student.GetInfo()) // Ahmet (20 yaşında)
```

#### Interface (Soyutlama)

```go
// Interface tanımlama
type Speaker interface {
    Speak() string
}

// Student struct'ı Speaker interface'ini implement ediyor
func (s *Student) Speak() string {
    return "Merhaba, ben " + s.Name
}

// Kullanım
var speaker Speaker = &Student{Name: "Ahmet"}
fmt.Println(speaker.Speak())
```

#### Error Handling (Hata Yönetimi)

```go
import "errors"

func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("sıfıra bölme")
    }
    return a / b, nil
}

// Kullanım - ALWAYS check errors!
result, err := divide(10, 0)
if err != nil {
    fmt.Println("HATA:", err)
    return
}
fmt.Println("Sonuç:", result)
```

#### Pointer (Referans)

```go
func updateAge(s *Student) {
    s.Age = 25  // Pointer ile orijinal değeri değiştiriyoruz
}

student := Student{Age: 20}
updateAge(&student)  // & ile adres gönderiyoruz
fmt.Println(student.Age) // 25 (değişti!)
```

### 1.3. Go Module (Paket Yönetimi)

```bash
# Yeni proje başlatma
go mod init nexpaces-api

# Paket yükleme
go get github.com/gofiber/fiber/v2

# Tüm bağımlılıkları indirme
go mod download

# Kullanılmayan paketleri temizleme
go mod tidy
```

### 1.4. Önemli Go Komutları

```bash
# Çalıştırma
go run cmd/server/main.go

# Build (binary oluşturma)
go build -o bin/server ./cmd/server

# Test çalıştırma
go test ./...

# Belirli bir paketi test etme
go test ./internal/domain/project -v

# Kod formatlama
go fmt ./...

# Linting (kod kalitesi kontrolü)
go vet ./...

# Bağımlılıkları gösterme
go list -m all
```

---

## 2. Backend Mimarisi Nedir?

### 2.1. Clean Architecture (NexSpaces'in kullandığı mimari)

```
┌──────────────────────────────────────────────┐
│         HTTP Layer (Handlers)                │
│  Kullanıcıdan gelen istekleri karşılar       │
└──────────────┬───────────────────────────────┘
               │
┌──────────────▼───────────────────────────────┐
│         UseCase Layer (Services)             │
│  Business logic (iş kuralları) burada        │
└──────────────┬───────────────────────────────┘
               │
┌──────────────▼───────────────────────────────┐
│         Repository Layer                     │
│  Database işlemleri (CRUD operations)        │
└──────────────┬───────────────────────────────┘
               │
┌──────────────▼───────────────────────────────┐
│         Domain Layer (Entities)              │
│  Core business entities (Student, Lesson)    │
└──────────────────────────────────────────────┘
```

**Neden bu mimari?**
- ✅ Test edilebilir (her katman bağımsız)
- ✅ Değiştirilebilir (database değişirse sadece repository değişir)
- ✅ Anlaşılır (her katmanın görevi belli)

### 2.2. Katmanların Görevleri

#### Domain Layer (Entities)
```go
// internal/domain/lesson/student_entity.go
type Student struct {
    ID        string
    FirstName string
    LastName  string
    Email     string
}

// Business rule (iş kuralı)
func (s *Student) Validate() error {
    if s.Email == "" {
        return errors.New("email gerekli")
    }
    return nil
}
```
**Görev:** Core business logic ve validasyon

#### Repository Layer
```go
// internal/repository/lesson/student_postgres.go
type StudentRepository interface {
    Create(ctx context.Context, student *Student) error
    GetByID(ctx context.Context, id string) (*Student, error)
}

func (r *PostgresRepo) Create(ctx context.Context, s *Student) error {
    query := "INSERT INTO students (id, first_name, email) VALUES ($1, $2, $3)"
    _, err := r.db.ExecContext(ctx, query, s.ID, s.FirstName, s.Email)
    return err
}
```
**Görev:** Database ile konuşur (SQL queries)

#### UseCase/Service Layer
```go
// internal/usecase/lesson/student_service.go
type StudentService struct {
    repo StudentRepository
}

func (s *StudentService) RegisterStudent(req *CreateStudentRequest) (*Student, error) {
    // 1. Validate
    if req.Email == "" {
        return nil, errors.New("email gerekli")
    }

    // 2. Business logic
    student := &Student{
        ID:    uuid.New().String(),
        Email: req.Email,
    }

    // 3. Save to DB
    if err := s.repo.Create(context.Background(), student); err != nil {
        return nil, err
    }

    return student, nil
}
```
**Görev:** Business logic ve orchestration

#### Handler Layer (HTTP)
```go
// internal/handler/lesson/student_handler.go
func (h *Handler) CreateStudent(c *fiber.Ctx) error {
    var req CreateStudentRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid body"})
    }

    student, err := h.service.RegisterStudent(&req)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.Status(201).JSON(student)
}
```
**Görev:** HTTP request/response yönetimi

---

## 3. Proje Klasör Yapısı Açıklaması

```
nexpaces-api/
│
├── cmd/
│   └── server/
│       └── main.go              # 🚀 Program buradan başlar (entry point)
│
├── internal/                    # Private kod (sadece bu projede kullanılır)
│   │
│   ├── domain/                  # 🧠 Core business entities
│   │   ├── project/
│   │   │   ├── entity.go        # Project struct tanımı
│   │   │   ├── errors.go        # Project için error tanımları
│   │   │   └── repository.go    # Repository interface (sözleşme)
│   │   │
│   │   ├── lesson/
│   │   │   ├── student_entity.go   # Student struct
│   │   │   ├── lesson_entity.go    # Lesson struct
│   │   │   ├── assignment_entity.go # Assignment struct
│   │   │   ├── errors.go
│   │   │   └── repository.go
│   │   │
│   │   ├── user/                # User entity
│   │   ├── tenant/              # Tenant entity
│   │   └── website/             # Website entity
│   │
│   ├── repository/              # 💾 Database işlemleri
│   │   ├── project/
│   │   │   └── postgres.go      # PostgreSQL implementation
│   │   ├── lesson/
│   │   │   ├── student_postgres.go
│   │   │   ├── lesson_postgres.go
│   │   │   └── assignment_postgres.go
│   │   ├── user/
│   │   └── tenant/
│   │
│   ├── usecase/                 # 🎯 Business logic
│   │   ├── project/
│   │   │   ├── service.go       # Project business logic
│   │   │   └── dto.go           # Data Transfer Objects (API request/response)
│   │   ├── lesson/
│   │   │   ├── student_service.go
│   │   │   ├── lesson_service.go
│   │   │   └── dto.go
│   │   ├── user/
│   │   └── tenant/
│   │
│   ├── handler/                 # 🌐 HTTP endpoints
│   │   ├── project/
│   │   │   └── handler.go       # Project API endpoints
│   │   ├── lesson/
│   │   │   ├── student_handler.go
│   │   │   ├── lesson_handler.go
│   │   │   └── assignment_handler.go
│   │   ├── auth/
│   │   ├── user/
│   │   └── tenant/
│   │
│   ├── middleware/              # 🛡️ HTTP middleware (auth, logging, etc.)
│   │   ├── auth.go              # JWT doğrulama
│   │   ├── tenant.go            # Tenant context
│   │   └── logger.go            # Request logging
│   │
│   ├── config/                  # ⚙️ Configuration
│   │   └── config.go            # Environment variables
│   │
│   └── app/                     # 🔧 Application bootstrap
│       ├── app.go               # Dependency injection (servisleri başlatır)
│       └── routes.go            # Route registration (URL'leri tanımlar)
│
├── pkg/                         # Public paketler (başka projelerde kullanılabilir)
│   ├── database/
│   │   └── postgres.go          # PostgreSQL bağlantı helper
│   ├── logger/
│   │   └── logger.go            # Structured logging
│   ├── validator/
│   │   └── validator.go         # Custom validators
│   └── errors/
│       └── errors.go            # Error utilities
│
├── migrations/                  # 📊 Database migrations (tablo oluşturma scriptleri)
│   ├── 001_create_tenants.up.sql
│   ├── 001_create_tenants.down.sql
│   ├── 002_create_users.up.sql
│   ├── 003_create_websites.up.sql
│   ├── 004_create_projects.up.sql
│   ├── 005_create_students.up.sql
│   ├── 006_create_lessons.up.sql
│   └── 007_create_assignments.up.sql
│
├── test/                        # 🧪 Test helpers
│   └── helpers/
│       ├── database.go          # Test database setup
│       └── fixtures.go          # Test data oluşturma
│
├── docker/
│   └── Dockerfile               # Docker image tanımı
│
├── go.mod                       # Go module tanımı (package.json gibi)
├── go.sum                       # Dependency checksums
├── docker-compose.yml           # Local development stack
├── Makefile                     # Development komutları
├── .env.example                 # Environment variables örneği
└── README.md                    # Proje dökümanı
```

### Dosya Türleri ve Görevleri

| Dosya | Görev | Örnek |
|-------|-------|-------|
| `entity.go` | Struct tanımı ve business rules | Student, Lesson |
| `repository.go` | Database interface tanımı | Create, GetByID, List |
| `*_postgres.go` | PostgreSQL implementation | SQL queries |
| `service.go` | Business logic | RegisterStudent, ScheduleLesson |
| `dto.go` | API request/response yapıları | CreateStudentRequest |
| `handler.go` | HTTP endpoints | POST /api/v1/students |
| `errors.go` | Error tanımları | ErrStudentNotFound |
| `*.up.sql` | Database tablo oluşturma | CREATE TABLE students |
| `*.down.sql` | Database tablo silme | DROP TABLE students |

---

## 4. Database (Veritabanı) Kullanımı

### 4.1. PostgreSQL Nedir?

PostgreSQL, güçlü, açık kaynaklı bir **ilişkisel veritabanı**dır.

**Temel Kavramlar:**
- **Table (Tablo):** Excel sheet gibi. Örnek: `students` tablosu
- **Row (Satır):** Bir kayıt. Örnek: Bir öğrenci
- **Column (Sütun):** Bir özellik. Örnek: `first_name`, `email`
- **Primary Key:** Benzersiz ID. Örnek: `id UUID PRIMARY KEY`
- **Foreign Key:** Başka tabloya referans. Örnek: `student_id REFERENCES students(id)`

### 4.2. Temel SQL Komutları

#### SELECT (Veri okuma)
```sql
-- Tüm öğrencileri getir
SELECT * FROM students;

-- Sadece isim ve email getir
SELECT first_name, email FROM students;

-- Filtreleme
SELECT * FROM students WHERE status = 'active';

-- Sıralama
SELECT * FROM students ORDER BY created_at DESC;

-- Limit
SELECT * FROM students LIMIT 10;
```

#### INSERT (Veri ekleme)
```sql
INSERT INTO students (id, first_name, last_name, email, status)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'Ahmet',
    'Yılmaz',
    'ahmet@example.com',
    'active'
);
```

#### UPDATE (Veri güncelleme)
```sql
UPDATE students
SET status = 'graduated'
WHERE id = '550e8400-e29b-41d4-a716-446655440000';
```

#### DELETE (Veri silme)
```sql
-- Hard delete (gerçek silme)
DELETE FROM students WHERE id = 'xxx';

-- Soft delete (silindi olarak işaretle) - ÖNERİLİR
UPDATE students SET deleted_at = NOW() WHERE id = 'xxx';
```

### 4.3. Migration Nedir?

Migration, veritabanı değişikliklerini **versiyonlamak** için kullanılır.

**Örnek Migration:**
```sql
-- migrations/005_create_students.up.sql (Tablo oluştur)
CREATE TABLE students (
    id UUID PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);
```

```sql
-- migrations/005_create_students.down.sql (Geri al)
DROP TABLE students;
```

**Migration komutları:**
```bash
# Migration çalıştır (yukarı)
make migrate-up

# Migration geri al (aşağı)
make migrate-down

# Migration durumu
make migrate-status
```

### 4.4. PostgreSQL Docker ile Çalıştırma

```bash
# PostgreSQL başlat
docker-compose up -d postgres

# PostgreSQL'e bağlan (psql cli)
docker exec -it nexspaces-postgres psql -U postgres -d nexspaces_dev

# Database içinde:
\dt              # Tabloları listele
\d students      # students tablosunu incele
\q               # Çık
```

**Yaygın psql komutları:**
```sql
\l               -- Database'leri listele
\c nexspaces_dev -- Database'e geç
\dt              -- Tabloları listele
\d students      -- Tablo yapısını göster
\du              -- Kullanıcıları listele
```

### 4.5. Go'da PostgreSQL Kullanımı

```go
import (
    "database/sql"
    _ "github.com/lib/pq"
)

// Bağlantı oluşturma
db, err := sql.Open("postgres",
    "host=localhost port=5432 user=postgres password=postgres dbname=nexspaces_dev sslmode=disable")
if err != nil {
    panic(err)
}
defer db.Close()

// Query (okuma)
rows, err := db.Query("SELECT id, first_name, email FROM students")
if err != nil {
    panic(err)
}
defer rows.Close()

for rows.Next() {
    var id, firstName, email string
    rows.Scan(&id, &firstName, &email)
    fmt.Println(id, firstName, email)
}

// Exec (yazma)
_, err = db.Exec("INSERT INTO students (id, first_name, email) VALUES ($1, $2, $3)",
    uuid.New().String(), "Ahmet", "ahmet@example.com")
```

---

## 5. Docker Kullanımı

### 5.1. Docker Nedir?

Docker, uygulamaları **container** (kapsayıcı) içinde çalıştırır. Container, her şeyi içinde barındıran izole bir ortamdır.

**Neden Docker?**
- ✅ "Benim bilgisayarda çalışıyor" problemi yok
- ✅ Hızlı kurulum (PostgreSQL, Redis tek komutla)
- ✅ Production ile aynı ortam

### 5.2. Temel Docker Komutları

```bash
# Container'ları başlat
docker-compose up -d

# Container'ları durdur
docker-compose down

# Container'ları göster
docker ps

# Tüm container'ları göster (durmuş olanlar dahil)
docker ps -a

# Container loglarını göster
docker-compose logs -f api

# Container içinde komut çalıştır
docker exec -it nexspaces-postgres psql -U postgres

# Image'leri listele
docker images

# Kullanılmayan kaynakları temizle
docker system prune -a
```

### 5.3. docker-compose.yml Açıklaması

```yaml
version: '3.8'

services:
  # PostgreSQL veritabanı
  postgres:
    image: postgres:15-alpine    # Kullanılacak image
    container_name: nexspaces-postgres
    environment:
      POSTGRES_USER: postgres     # Database kullanıcısı
      POSTGRES_PASSWORD: postgres # Şifre
      POSTGRES_DB: nexspaces_dev  # Database adı
    ports:
      - "5432:5432"               # Host:Container port mapping
    volumes:
      - postgres_data:/var/lib/postgresql/data  # Veri kalıcılığı
    healthcheck:                  # Container sağlık kontrolü
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  # API servisi
  api:
    build:
      context: .
      dockerfile: docker/Dockerfile
    container_name: nexpaces-api
    ports:
      - "8080:8080"
    environment:
      DATABASE_HOST: postgres     # PostgreSQL'e bağlan
      DATABASE_PORT: 5432
      DATABASE_USER: postgres
      DATABASE_PASSWORD: postgres
      DATABASE_NAME: nexspaces_dev
    depends_on:
      - postgres                  # PostgreSQL başlamadan API başlamasın
    volumes:
      - .:/app                    # Kod değişikliklerini otomatik yansıt

volumes:
  postgres_data:                  # Named volume (veri burada saklanır)
```

### 5.4. Dockerfile Açıklaması

```dockerfile
# Multi-stage build (küçük image için)

# Stage 1: Build
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download              # Bağımlılıkları indir
COPY . .
RUN go build -o bin/server ./cmd/server  # Binary oluştur

# Stage 2: Run
FROM alpine:latest
RUN apk --no-cache add ca-certificates  # SSL sertifikaları
WORKDIR /root/
COPY --from=builder /app/bin/server .   # Binary'yi kopyala
EXPOSE 8080
CMD ["./server"]                         # Uygulamayı başlat
```

---

## 6. Kalan Görevler ve Nasıl Yapılır

### ✅ Tamamlanan Modüller

- [x] **Projects Module** - Portfolio yönetimi (TAMAMLANDI)
- [x] **Lessons Module - Domain Layer** - Student, Lesson, Assignment entities (TAMAMLANDI)
- [x] **Lessons Module - Repository Layer** - Database işlemleri (TAMAMLANDI)

### 🔄 Devam Eden Görevler

#### Görev 1: Lessons Module - Service Layer (UseCase)

**Ne yapacaksın?**
Student, Lesson ve Assignment için business logic yazacaksın.

**Adımlar:**

1. **Student Service oluştur:**

```bash
# Dosya: internal/usecase/lesson/student_service.go
```

```go
package lesson

import (
    "context"
    "fmt"

    "nexpaces-api/internal/domain/lesson"
    "nexpaces-api/pkg/logger"
    "nexpaces-api/pkg/validator"
)

type StudentService struct {
    repo      lesson.StudentRepository
    validator *validator.Validator
    logger    *logger.Logger
}

func NewStudentService(repo lesson.StudentRepository, validator *validator.Validator, logger *logger.Logger) *StudentService {
    return &StudentService{
        repo:      repo,
        validator: validator,
        logger:    logger,
    }
}

func (s *StudentService) Create(ctx context.Context, tenantID string, req *CreateStudentRequest) (*StudentResponse, error) {
    // 1. Validate request
    if err := s.validator.Validate(req); err != nil {
        return nil, err
    }

    // 2. Create student entity
    student, err := lesson.NewStudent(
        tenantID,
        req.FirstName,
        req.LastName,
        req.Email,
        req.Phone,
        lesson.StudentLevel(req.Level),
    )
    if err != nil {
        return nil, err
    }

    // 3. Save to database
    if err := s.repo.Create(ctx, student); err != nil {
        s.logger.WithFields(logger.Fields{"tenant_id": tenantID, "error": err}).Error("failed to create student")
        return nil, fmt.Errorf("failed to create student: %w", err)
    }

    s.logger.WithFields(logger.Fields{"student_id": student.ID, "tenant_id": tenantID}).Info("student created successfully")

    return toStudentResponse(student), nil
}

func (s *StudentService) GetByID(ctx context.Context, id string) (*StudentResponse, error) {
    student, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    return toStudentResponse(student), nil
}

// ... diğer methodlar (List, Update, Delete, Suspend, Activate, Graduate)

func toStudentResponse(s *lesson.Student) *StudentResponse {
    return &StudentResponse{
        ID:               s.ID,
        TenantID:         s.TenantID,
        FirstName:        s.FirstName,
        LastName:         s.LastName,
        Email:            s.Email,
        Phone:            s.Phone,
        Status:           string(s.Status),
        Level:            string(s.Level),
        TotalLessons:     s.TotalLessons,
        CompletedLessons: s.CompletedLessons,
        AttendanceRate:   s.AttendanceRate,
        AverageScore:     s.AverageScore,
        CreatedAt:        s.CreatedAt,
        UpdatedAt:        s.UpdatedAt,
    }
}
```

2. **DTO (Data Transfer Objects) oluştur:**

```bash
# Dosya: internal/usecase/lesson/dto.go
```

```go
package lesson

import "time"

// Student DTOs
type CreateStudentRequest struct {
    FirstName   string   `json:"first_name" validate:"required,min=2,max=100"`
    LastName    string   `json:"last_name" validate:"required,min=2,max=100"`
    Email       string   `json:"email" validate:"required,email"`
    Phone       string   `json:"phone" validate:"required"`
    Level       string   `json:"level" validate:"required,oneof=beginner intermediate advanced"`
    Grade       string   `json:"grade,omitempty"`
    School      string   `json:"school,omitempty"`
    ParentName  string   `json:"parent_name,omitempty"`
    ParentEmail string   `json:"parent_email,omitempty" validate:"omitempty,email"`
    ParentPhone string   `json:"parent_phone,omitempty"`
}

type UpdateStudentRequest struct {
    FirstName *string `json:"first_name,omitempty" validate:"omitempty,min=2,max=100"`
    LastName  *string `json:"last_name,omitempty" validate:"omitempty,min=2,max=100"`
    Email     *string `json:"email,omitempty" validate:"omitempty,email"`
    Phone     *string `json:"phone,omitempty"`
    Level     *string `json:"level,omitempty" validate:"omitempty,oneof=beginner intermediate advanced"`
    Status    *string `json:"status,omitempty"`
    Grade     *string `json:"grade,omitempty"`
}

type StudentResponse struct {
    ID               string    `json:"id"`
    TenantID         string    `json:"tenant_id"`
    FirstName        string    `json:"first_name"`
    LastName         string    `json:"last_name"`
    Email            string    `json:"email"`
    Phone            string    `json:"phone"`
    Status           string    `json:"status"`
    Level            string    `json:"level"`
    Grade            string    `json:"grade,omitempty"`
    TotalLessons     int       `json:"total_lessons"`
    CompletedLessons int       `json:"completed_lessons"`
    AttendanceRate   float64   `json:"attendance_rate"`
    AverageScore     float64   `json:"average_score"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}

type ListStudentsRequest struct {
    Status    *string `json:"status,omitempty"`
    Level     *string `json:"level,omitempty"`
    Search    string  `json:"search,omitempty"`
    Page      int     `json:"page,omitempty" validate:"omitempty,min=1"`
    PageSize  int     `json:"page_size,omitempty" validate:"omitempty,min=1,max=100"`
}

type ListStudentsResponse struct {
    Students   []*StudentResponse `json:"students"`
    Total      int64              `json:"total"`
    Page       int                `json:"page"`
    PageSize   int                `json:"page_size"`
    TotalPages int                `json:"total_pages"`
}

// Lesson DTOs (benzer şekilde oluştur)
// Assignment DTOs (benzer şekilde oluştur)
```

**Nasıl test edersin?**
```bash
# Build kontrol
go build -o /dev/null ./...

# Eğer hata varsa düzelt
```

---

#### Görev 2: Lessons Module - Handler Layer (HTTP Endpoints)

**Ne yapacaksın?**
HTTP endpoints oluşturacaksın (API).

**Adımlar:**

1. **Student Handler oluştur:**

```bash
# Dosya: internal/handler/lesson/student_handler.go
```

```go
package lesson

import (
    "github.com/gofiber/fiber/v2"
    studentUsecase "nexpaces-api/internal/usecase/lesson"
)

type StudentHandler struct {
    service *studentUsecase.StudentService
}

func NewStudentHandler(service *studentUsecase.StudentService) *StudentHandler {
    return &StudentHandler{service: service}
}

// POST /api/v1/students
func (h *StudentHandler) Create(c *fiber.Ctx) error {
    tenantID := c.Locals("tenant_id").(string)

    var req studentUsecase.CreateStudentRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    student, err := h.service.Create(c.Context(), tenantID, &req)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "success": true,
        "data":    student,
    })
}

// GET /api/v1/students/:id
func (h *StudentHandler) GetByID(c *fiber.Ctx) error {
    id := c.Params("id")

    student, err := h.service.GetByID(c.Context(), id)
    if err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Student not found",
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data":    student,
    })
}

// GET /api/v1/students
func (h *StudentHandler) List(c *fiber.Ctx) error {
    var req studentUsecase.ListStudentsRequest
    if err := c.QueryParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid query parameters",
        })
    }

    response, err := h.service.List(c.Context(), &req)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data":    response,
    })
}

// PATCH /api/v1/students/:id
func (h *StudentHandler) Update(c *fiber.Ctx) error {
    id := c.Params("id")

    var req studentUsecase.UpdateStudentRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    student, err := h.service.Update(c.Context(), id, &req)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data":    student,
    })
}

// DELETE /api/v1/students/:id
func (h *StudentHandler) Delete(c *fiber.Ctx) error {
    id := c.Params("id")

    if err := h.service.Delete(c.Context(), id); err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "message": "Student deleted successfully",
    })
}

// POST /api/v1/students/:id/suspend
func (h *StudentHandler) Suspend(c *fiber.Ctx) error {
    id := c.Params("id")

    var req struct {
        Reason string `json:"reason" validate:"required"`
    }
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    student, err := h.service.Suspend(c.Context(), id, req.Reason)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data":    student,
    })
}

// POST /api/v1/students/:id/activate
func (h *StudentHandler) Activate(c *fiber.Ctx) error {
    id := c.Params("id")

    student, err := h.service.Activate(c.Context(), id)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data":    student,
    })
}

// POST /api/v1/students/:id/graduate
func (h *StudentHandler) Graduate(c *fiber.Ctx) error {
    id := c.Params("id")

    student, err := h.service.Graduate(c.Context(), id)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data":    student,
    })
}

// GET /api/v1/students/stats
func (h *StudentHandler) GetStats(c *fiber.Ctx) error {
    stats, err := h.service.GetStats(c.Context())
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data":    stats,
    })
}
```

2. **Benzer şekilde Lesson ve Assignment handler'larını oluştur**

---

#### Görev 3: App Bootstrap'a Ekle

**Ne yapacaksın?**
Oluşturduğun servisleri ve handler'ları uygulamaya kaydetme.

**Adımlar:**

1. **internal/app/app.go'yu güncelle:**

```go
// Student service & handler ekle
studentRepo := lessonRepo.NewStudentPostgresRepository(db)
studentService := lessonUsecase.NewStudentService(studentRepo, appValidator, appLogger)
studentHandler := lessonHandler.NewStudentHandler(studentService)

// Lesson service & handler ekle
lessonRepo := lessonRepo.NewLessonPostgresRepository(db)
lessonService := lessonUsecase.NewLessonService(lessonRepo, appValidator, appLogger)
lessonHandler := lessonHandler.NewLessonHandler(lessonService)

// Assignment service & handler ekle
assignmentRepo := lessonRepo.NewAssignmentPostgresRepository(db)
assignmentService := lessonUsecase.NewAssignmentService(assignmentRepo, appValidator, appLogger)
assignmentHandler := lessonHandler.NewAssignmentHandler(assignmentService)
```

2. **internal/app/routes.go'ya route ekle:**

```go
// Student routes
students := api.Group("/students")
students.Post("/", authMiddleware, studentHandler.Create)
students.Get("/", authMiddleware, studentHandler.List)
students.Get("/stats", authMiddleware, studentHandler.GetStats)
students.Get("/:id", authMiddleware, studentHandler.GetByID)
students.Patch("/:id", authMiddleware, studentHandler.Update)
students.Delete("/:id", authMiddleware, studentHandler.Delete)
students.Post("/:id/suspend", authMiddleware, studentHandler.Suspend)
students.Post("/:id/activate", authMiddleware, studentHandler.Activate)
students.Post("/:id/graduate", authMiddleware, studentHandler.Graduate)

// Lesson routes
lessons := api.Group("/lessons")
lessons.Post("/", authMiddleware, lessonHandler.Create)
lessons.Get("/", authMiddleware, lessonHandler.List)
lessons.Get("/upcoming", authMiddleware, lessonHandler.GetUpcoming)
lessons.Get("/:id", authMiddleware, lessonHandler.GetByID)
lessons.Patch("/:id", authMiddleware, lessonHandler.Update)
lessons.Delete("/:id", authMiddleware, lessonHandler.Delete)
lessons.Post("/:id/start", authMiddleware, lessonHandler.Start)
lessons.Post("/:id/complete", authMiddleware, lessonHandler.Complete)
lessons.Post("/:id/cancel", authMiddleware, lessonHandler.Cancel)

// Assignment routes
assignments := api.Group("/assignments")
assignments.Post("/", authMiddleware, assignmentHandler.Create)
assignments.Get("/", authMiddleware, assignmentHandler.List)
assignments.Get("/overdue", authMiddleware, assignmentHandler.GetOverdue)
assignments.Get("/:id", authMiddleware, assignmentHandler.GetByID)
assignments.Patch("/:id", authMiddleware, assignmentHandler.Update)
assignments.Delete("/:id", authMiddleware, assignmentHandler.Delete)
assignments.Post("/:id/submit", authMiddleware, assignmentHandler.Submit)
assignments.Post("/:id/grade", authMiddleware, assignmentHandler.Grade)
```

---

#### Görev 4: Diğer Modülleri Tamamla

**Sıra:**
1. ✅ Projects (TAMAMLANDI)
2. ✅ Lessons (Devam ediyor - service/handler eklenecek)
3. ⏳ Bookings/Calendar
4. ⏳ Services/Pricing
5. ⏳ Blog/CMS

**Her modül için aynı adımları izle:**
1. Domain layer (entity, errors, repository interface)
2. Repository layer (postgres implementation)
3. UseCase layer (service + dto)
4. Handler layer (HTTP endpoints)
5. App bootstrap'a ekle

---

## 7. Hata Ayıklama (Debugging)

### 7.1. Build Hataları

```bash
# Build hatalarını gör
go build ./...

# Yaygın hatalar:
# - "undefined: XXX" → Package import eksik
# - "cannot use X as Y" → Tip uyumsuzluğu
# - "declared but not used" → Kullanılmayan değişken
```

### 7.2. Runtime Hataları

```bash
# Uygulamayı çalıştır ve logları izle
go run cmd/server/main.go

# Docker loglarını izle
docker-compose logs -f api
```

### 7.3. Database Hataları

```bash
# PostgreSQL loglarını izle
docker-compose logs -f postgres

# Database'e bağlan ve kontrol et
docker exec -it nexspaces-postgres psql -U postgres -d nexspaces_dev

# Tablo var mı?
\dt

# Veri var mı?
SELECT * FROM students LIMIT 5;
```

### 7.4. API Test Etme (Postman/cURL)

```bash
# Health check
curl http://localhost:8080/health

# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "first_name": "Test",
    "last_name": "User"
  }'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'

# Student oluştur (JWT token ile)
curl -X POST http://localhost:8080/api/v1/students \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "first_name": "Ahmet",
    "last_name": "Yılmaz",
    "email": "ahmet@example.com",
    "phone": "+905551234567",
    "level": "beginner"
  }'
```

---

## 8. Test Yazma

### 8.1. Unit Test (Domain Layer)

```go
// internal/domain/lesson/student_entity_test.go
package lesson

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestNewStudent(t *testing.T) {
    // Test case 1: Valid student
    student, err := NewStudent(
        "tenant-123",
        "Ahmet",
        "Yılmaz",
        "ahmet@example.com",
        "+905551234567",
        LevelBeginner,
    )

    assert.NoError(t, err)
    assert.NotNil(t, student)
    assert.Equal(t, "Ahmet", student.FirstName)
    assert.Equal(t, StudentStatusActive, student.Status)

    // Test case 2: Invalid email
    _, err = NewStudent("tenant-123", "Ahmet", "Yılmaz", "", "+90555", LevelBeginner)
    assert.Error(t, err)
}

func TestStudent_Suspend(t *testing.T) {
    student, _ := NewStudent("tenant-123", "Ahmet", "Yılmaz", "ahmet@example.com", "+90555", LevelBeginner)

    err := student.Suspend("Not paying")
    assert.NoError(t, err)
    assert.Equal(t, StudentStatusSuspended, student.Status)

    // Already suspended
    err = student.Suspend("Another reason")
    assert.Error(t, err)
    assert.Equal(t, ErrStudentAlreadySuspended, err)
}
```

**Test çalıştırma:**
```bash
# Tüm testler
go test ./...

# Verbose mode
go test -v ./internal/domain/lesson

# Coverage
go test -cover ./internal/domain/lesson
```

---

## 9. Deployment (Canlıya Alma)

### 9.1. Production Build

```bash
# Binary oluştur
go build -o bin/server ./cmd/server

# Docker image oluştur
docker build -f docker/Dockerfile -t nexpaces-api:latest .

# Image'i çalıştır
docker run -p 8080:8080 \
  -e DATABASE_HOST=your-db-host \
  -e DATABASE_PASSWORD=your-db-password \
  nexpaces-api:latest
```

### 9.2. Environment Variables (Production)

```bash
# .env (production)
SERVER_ENVIRONMENT=production
DATABASE_HOST=your-rds-endpoint.amazonaws.com
DATABASE_PASSWORD=strong-password-here
AUTH_JWT_SECRET=very-secure-secret-min-32-chars
SECURITY_ALLOWED_ORIGINS=https://nexpaces.com
```

### 9.3. Migration (Production)

```bash
# Production database'e migration çalıştır
DATABASE_URL="postgres://user:pass@host:5432/dbname?sslmode=require" \
  make migrate-up
```

---

## 🎯 Öncelikli Görev Listesi (TODO)

### Sprint 1: Lessons Module Tamamlama (2-3 gün)
- [ ] Student Service yazma (dto.go + service.go)
- [ ] Student Handler yazma (handler.go)
- [ ] Lesson Service yazma
- [ ] Lesson Handler yazma
- [ ] Assignment Service yazma
- [ ] Assignment Handler yazma
- [ ] App bootstrap'a ekleme (app.go + routes.go)
- [ ] Build kontrol: `go build ./...`
- [ ] Postman ile test etme
- [ ] Commit: "feat: complete Lessons module service and handler layers"

### Sprint 2: Bookings/Calendar Module (3-4 gün)
- [ ] Domain layer (availability, appointment entities)
- [ ] Repository layer
- [ ] Service layer
- [ ] Handler layer
- [ ] Calendar integration (iCal export)

### Sprint 3: Services/Pricing Module (2-3 gün)
- [ ] Domain layer (service, pricing entities)
- [ ] Repository layer
- [ ] Service layer
- [ ] Handler layer

### Sprint 4: Blog/CMS Module (3-4 gün)
- [ ] Domain layer (post, category, tag entities)
- [ ] Repository layer
- [ ] Service layer
- [ ] Handler layer
- [ ] Rich text editor support

### Sprint 5: Integration & Testing (1 hafta)
- [ ] End-to-end testing
- [ ] Multi-tenant testing (canakyuz.co tenant)
- [ ] API documentation (Swagger)
- [ ] Performance testing

### Sprint 6: Deployment (3-5 gün)
- [ ] AWS/Railway setup
- [ ] Production database
- [ ] CI/CD pipeline
- [ ] Domain configuration
- [ ] SSL certificates

---

## 📖 Öğrenme Kaynakları

### Go Öğrenme
- **Tour of Go:** https://go.dev/tour/ (Başla buradan!)
- **Go by Example:** https://gobyexample.com/
- **Effective Go:** https://go.dev/doc/effective_go

### Backend Concepts
- **Clean Architecture:** https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html
- **REST API Best Practices:** https://restfulapi.net/

### PostgreSQL
- **PostgreSQL Tutorial:** https://www.postgresqltutorial.com/
- **SQL Basics:** https://www.w3schools.com/sql/

### Docker
- **Docker Get Started:** https://docs.docker.com/get-started/
- **Docker Compose:** https://docs.docker.com/compose/

---

## 🆘 Yardım

### Sık Sorulan Sorular

**S: Build hatası alıyorum, ne yapmalıyım?**
```bash
# 1. Dependencies'i güncelle
go mod tidy

# 2. Cache temizle
go clean -modcache

# 3. Tekrar build et
go build ./...
```

**S: Database'e bağlanamıyorum**
```bash
# PostgreSQL çalışıyor mu?
docker ps | grep postgres

# Çalışmıyorsa başlat
docker-compose up -d postgres

# Logları kontrol et
docker-compose logs postgres
```

**S: Migration hatası alıyorum**
```bash
# Migration durumunu kontrol et
make migrate-status

# Force rollback
make migrate-down
make migrate-up
```

**S: JWT token nasıl alırım?**
```bash
# 1. Register ol
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"test123","first_name":"Test","last_name":"User"}'

# 2. Login yap
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"test123"}'

# Response'da "token" field'ı var
```

---

## ✅ Checklist - Her Modül İçin

Bir modül tamamlandığında kontrol et:

- [ ] Domain entities var mı? (entity.go, errors.go, repository.go)
- [ ] Repository implementation var mı? (*_postgres.go)
- [ ] Migration dosyaları var mı? (XXX_create_table.up.sql, .down.sql)
- [ ] Service/UseCase var mı? (service.go, dto.go)
- [ ] Handler var mı? (handler.go)
- [ ] App bootstrap'a eklendi mi? (app.go, routes.go)
- [ ] Build çalışıyor mu? (`go build ./...`)
- [ ] Database migration çalıştı mı? (`make migrate-up`)
- [ ] API test edildi mi? (Postman/cURL)
- [ ] Commit atıldı mı?

---

## 🎓 Sonuç

Bu rehberi takip ederek:
1. ✅ Go dilini öğreneceksin
2. ✅ Backend mimarisini anlayacaksın
3. ✅ Database kullanmayı öğreneceksin
4. ✅ Docker kullanmayı öğreneceksin
5. ✅ NexSpaces projesini tamamlayacaksın

**Başarılar! 🚀**

Her sorunda bu rehbere dön. Takıldığın yerde kaldığın yerden devam et.

---

**Hazırlayan:** Claude Code
**Tarih:** 1 Ekim 2025
**Versiyon:** 1.0
