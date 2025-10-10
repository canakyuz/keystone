package database

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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
func TestExecuteInTenantContext(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewTenantConnectionManager(db, nil)

	t.Run("başarılı execution", func(t *testing.T) {
		schemaName := "tenant_acme"

		// Mock beklentileri:
		// 1. Connection al
		mock.ExpectBegin() // Connection hazır

		// 2. SET search_path
		mock.ExpectExec(`SET search_path TO "tenant_acme", public`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// 3. User function'ın içindeki query
		mock.ExpectQuery("SELECT COUNT").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		// 4. RESET search_path (defer)
		mock.ExpectExec(`SET search_path TO public`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectCommit() // Connection kapatıldı

		// Test fonksiyonu
		var queryResult int
		err := manager.ExecuteInTenantContext(context.Background(), schemaName, func() error {
			// Bu fonksiyon tenant_acme schema'sında çalışır
			row := db.QueryRow("SELECT COUNT(*) FROM users")
			return row.Scan(&queryResult)
		})

		assert.NoError(t, err)
		assert.Equal(t, 5, queryResult)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("function içinde hata oluşursa", func(t *testing.T) {
		schemaName := "tenant_test"

		mock.ExpectBegin()
		mock.ExpectExec(`SET search_path TO "tenant_test", public`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// defer'daki reset yine de çalışmalı
		mock.ExpectExec(`SET search_path TO public`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectCommit()

		// Function içinde kasıtlı hata üret
		testErr := sql.ErrNoRows
		err := manager.ExecuteInTenantContext(context.Background(), schemaName, func() error {
			return testErr
		})

		// Hata propagate edilmeli
		assert.Error(t, err)
		assert.Equal(t, testErr, err)

		// Reset yine de çalışmış olmalı
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("geçersiz schema adı", func(t *testing.T) {
		// Mock beklentisi YOK (validation fail olacak)

		err := manager.ExecuteInTenantContext(context.Background(), "invalid_schema", func() error {
			return nil
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "geçersiz schema adı")
	})
}

// TestGetConnection, connection pool'dan bağlantı almayı test eder.
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
func BenchmarkExecuteInTenantContext(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	manager := NewTenantConnectionManager(db, nil)
	schemaName := "tenant_bench"

	// Mock setup (her iteration için)
	for i := 0; i < b.N; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SET search_path TO "tenant_bench", public`).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(`SET search_path TO public`).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}

	b.ResetTimer() // Setup süresini sayma

	for i := 0; i < b.N; i++ {
		_ = manager.ExecuteInTenantContext(context.Background(), schemaName, func() error {
			// Minimal work
			return nil
		})
	}
}
