package database

import (
	"context"
	"database/sql"
	"testing"

	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	pq "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateSchemaName, schema adı validasyonunu test eder.
//
// ÖĞRENİLECEKLER:
// - Table-driven test pattern
// - Pozitif ve negatif test senaryoları
// - Security validation testing
func TestValidateSchemaName(t *testing.T) {
	tests := []struct {
		name        string
		schemaName  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "geçerli schema adı",
			schemaName:  "tenant_acme",
			expectError: false,
		},
		{
			name:        "rakam içeren geçerli schema",
			schemaName:  "tenant_acme_123",
			expectError: false,
		},
		{
			name:        "boş schema adı",
			schemaName:  "",
			expectError: true,
			errorMsg:    "1-63 karakter",
		},
		{
			name:        "tenant_ prefix olmayan",
			schemaName:  "acme_tenant",
			expectError: true,
			errorMsg:    "tenant_",
		},
		{
			name:        "SQL injection denemesi",
			schemaName:  "tenant_acme; DROP TABLE users;",
			expectError: true,
			errorMsg:    "geçersiz karakter",
		},
		{
			name:        "system schema erişimi",
			schemaName:  "pg_catalog",
			expectError: true,
			errorMsg:    "tenant_",
		},
		{
			name:        "büyük harf içeren",
			schemaName:  "tenant_Acme",
			expectError: true,
			errorMsg:    "geçersiz karakter",
		},
		{
			name:        "çok uzun schema adı",
			schemaName:  "tenant_" + string(make([]byte, 100)),
			expectError: true,
			errorMsg:    "1-63 karakter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSchemaName(tt.schemaName)

			if tt.expectError {
				assert.Error(t, err, "Hata beklendi ama nil döndü")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err, "Hata beklenmiyordu: %v", err)
			}
		})
	}
}

// TestSetSearchPath, search_path set işlemini test eder.
//
// ÖĞRENİLECEKLER:
// - sqlmock kullanarak database'i mock'lama
// - SQL query beklentilerini tanımlama
// - Error handling test etme
func TestSetSearchPath(t *testing.T) {
	// sqlmock ile fake database oluştur
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "sqlmock oluşturulamadı")
	defer db.Close()

	// Logger nil olabilir (test ortamında)
	manager := NewTenantConnectionManager(db, nil)

	t.Run("başarılı search_path set", func(t *testing.T) {
		schemaName := "tenant_acme"

		// Beklenen SQL komutu
		expectedSQL := `SET search_path TO "tenant_acme", public`

		// Mock: Bu SQL'in çalışacağını söyle
		mock.ExpectExec(expectedSQL).WillReturnResult(sqlmock.NewResult(0, 0))

		// Test et
		err := manager.SetSearchPath(context.Background(), schemaName)

		// Assertions
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL beklentileri karşılanmadı")
	})

	t.Run("geçersiz schema adı", func(t *testing.T) {
		// SQL injection denemesi
		schemaName := "tenant_acme; DROP TABLE users;"

		// Mock beklentisi YOK (SQL çalışmamalı çünkü validation fail olacak)

		err := manager.SetSearchPath(context.Background(), schemaName)

		// Validation error bekliyoruz
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "geçersiz")
	})

	t.Run("database hatası", func(t *testing.T) {
		schemaName := "tenant_test"
		expectedSQL := `SET search_path TO "tenant_test", public`

		// Mock: SQL çalışacak ama hata dönecek
		mock.ExpectExec(expectedSQL).WillReturnError(sql.ErrConnDone)

		err := manager.SetSearchPath(context.Background(), schemaName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "search_path ayarlanamadı")
	})
}

// TestResetSearchPath, search_path reset işlemini test eder.
func TestResetSearchPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewTenantConnectionManager(db, nil)

	t.Run("başarılı reset", func(t *testing.T) {
		expectedSQL := `SET search_path TO public`
		mock.ExpectExec(expectedSQL).WillReturnResult(sqlmock.NewResult(0, 0))

		err := manager.ResetSearchPath(context.Background())

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestExecuteInTenantContext, tenant context içinde fonksiyon çalıştırmayı test eder.
//
// ÖĞRENİLECEKLER:
// - Connection pool mock'lama
// - defer ile cleanup test etme
// - Error propagation test etme
// TestExecuteInTenantContext, gerçek PostgreSQL üzerinde çalışır.
//
// NEDEN mock değil: bu testin doğrulamak istediği şey, callback'in aldığı
// bağlantıda tenant schema'sının GERÇEKTEN aktif olması. sqlmock sorguyu
// yorumlamaz, yalnızca metin eşleştirir; search_path'in etkili olup olmadığını
// gösteremez. Nitekim bu testin mock'lu hali, uygulama havuz üzerinden sorgu
// çalıştırdığı halde geçiyordu.
func TestExecuteInTenantContext(t *testing.T) {
	db := openTestDB(t)
	if db == nil {
		t.Skip("postgres erişilemiyor")
	}
	ctx := context.Background()
	manager := NewTenantConnectionManager(db, nil)

	// İki şemada aynı isimli tablo: hangi şemanın aktif olduğunu ayırt etmek için.
	for schema, value := range map[string]int{"tenant_alpha": 1, "tenant_beta": 2} {
		mustExec(t, db, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", pq.QuoteIdentifier(schema)))
		mustExec(t, db, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.marker (id int)", pq.QuoteIdentifier(schema)))
		mustExec(t, db, fmt.Sprintf("TRUNCATE %s.marker", pq.QuoteIdentifier(schema)))
		mustExec(t, db, fmt.Sprintf("INSERT INTO %s.marker VALUES (%d)", pq.QuoteIdentifier(schema), value))
	}

	t.Run("callback tenant schemasinda calisir", func(t *testing.T) {
		var got int
		err := manager.ExecuteInTenantContext(ctx, "tenant_alpha", func(conn *sql.Conn) error {
			return conn.QueryRowContext(ctx, "SELECT id FROM marker").Scan(&got)
		})
		require.NoError(t, err)
		assert.Equal(t, 1, got, "alpha semasindaki satir okunmali")
	})

	t.Run("farkli tenant farkli veri gorur", func(t *testing.T) {
		var got int
		err := manager.ExecuteInTenantContext(ctx, "tenant_beta", func(conn *sql.Conn) error {
			return conn.QueryRowContext(ctx, "SELECT id FROM marker").Scan(&got)
		})
		require.NoError(t, err)
		assert.Equal(t, 2, got, "beta semasindaki satir okunmali")
	})

	t.Run("callback hatasi yukari tasinir", func(t *testing.T) {
		sentinel := errors.New("is mantigi hatasi")
		err := manager.ExecuteInTenantContext(ctx, "tenant_alpha", func(conn *sql.Conn) error {
			return sentinel
		})
		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("gecersiz schema adi reddedilir", func(t *testing.T) {
		called := false
		err := manager.ExecuteInTenantContext(ctx, "public; DROP TABLE users", func(conn *sql.Conn) error {
			called = true
			return nil
		})
		assert.Error(t, err)
		assert.False(t, called, "gecersiz schema'da callback calistirilmamali")
	})

	t.Run("islem bitince search_path public'e doner", func(t *testing.T) {
		err := manager.ExecuteInTenantContext(ctx, "tenant_alpha", func(conn *sql.Conn) error {
			return nil
		})
		require.NoError(t, err)

		// Havuzdan yeni bir baglanti al ve kirlenmemis oldugunu dogrula.
		var path string
		require.NoError(t, db.QueryRowContext(ctx, "SHOW search_path").Scan(&path))
		assert.NotContains(t, path, "tenant_alpha", "search_path havuza sizdi")
	})
}

func TestGetConnection(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewTenantConnectionManager(db, nil)

	t.Run("başarılı connection alma", func(t *testing.T) {
		conn, err := manager.GetConnection(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, conn)

		// Connection'ı kapat (pool'a geri dön)
		conn.Close()
	})

	t.Run("iptal edilmiş context", func(t *testing.T) {
		// Önceden iptal edilmiş context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Hemen iptal et

		conn, err := manager.GetConnection(ctx)

		assert.Error(t, err)
		assert.Nil(t, conn)
	})
}

// BenchmarkExecuteInTenantContext, performans testi.
//
// ÖĞRENİLECEKLER:
// - Benchmark yazma
// - Performance profiling
// - Memory allocation ölçme
// BenchmarkExecuteInTenantContext, tenant context'e girip çıkmanın maliyetini
// gerçek bir bağlantı havuzu üzerinde ölçer. Ölçülen şey bağlantı ayırma +
// iki SET search_path komutudur.
func BenchmarkExecuteInTenantContext(b *testing.B) {
	db := openBenchDB(b)
	if db == nil {
		b.Skip("postgres erişilemiyor")
	}
	ctx := context.Background()
	manager := NewTenantConnectionManager(db, nil)

	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS tenant_bench`); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.ExecuteInTenantContext(ctx, "tenant_bench", func(conn *sql.Conn) error {
			return nil
		})
	}
}
