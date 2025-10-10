package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/lib/pq"

	"nexpaces-api/pkg/logger"
)

// TenantConnectionManager, schema-per-tenant mimarisinde tenant izolasyonu sağlar.
//
// PostgreSQL'de search_path, sorguların hangi schema'da çalışacağını belirler.
// Bu manager, her request için doğru tenant schema'sının set edilmesini ve
// işlem sonrası temizlenmesini garanti eder.
type TenantConnectionManager interface {
	// SetSearchPath belirtilen tenant schema'sını aktif eder.
	// UYARI: Mutlaka ResetSearchPath ile temizlenmelidir (defer kullan).
	SetSearchPath(ctx context.Context, schemaName string) error

	// ResetSearchPath search_path'i varsayılan değere (public) döndürür.
	// Cross-tenant veri sızıntısını önlemek için kritik bir güvenlik adımıdır.
	ResetSearchPath(ctx context.Context) error

	// ExecuteInTenantContext bir fonksiyonu tenant schema context'i içinde çalıştırır.
	// Otomatik set/reset yaptığı için ÖNERİLEN kullanım budur.
	//
	// Örnek:
	//   err := manager.ExecuteInTenantContext(ctx, "tenant_acme", func() error {
	//       return userRepo.Create(ctx, user) // tenant_acme schema'sında çalışır
	//   })
	ExecuteInTenantContext(ctx context.Context, schemaName string, fn func() error) error

	// GetConnection connection pool'dan yeni bir bağlantı alır.
	// İleri düzey senaryolar için. Çoğu durumda ExecuteInTenantContext kullan.
	GetConnection(ctx context.Context) (*sql.Conn, error)
}

// connectionManager, TenantConnectionManager'ın implementasyonudur.
type connectionManager struct {
	db     *sql.DB
	logger *logger.Logger
	mu     sync.RWMutex
}

// NewTenantConnectionManager yeni bir TenantConnectionManager oluşturur.
func NewTenantConnectionManager(db *sql.DB, log *logger.Logger) TenantConnectionManager {
	return &connectionManager{
		db:     db,
		logger: log,
	}
}

// SetSearchPath, database bağlantısının search_path'ini değiştirir.
func (m *connectionManager) SetSearchPath(ctx context.Context, schemaName string) error {
	if err := validateSchemaName(schemaName); err != nil {
		return fmt.Errorf("geçersiz schema adı: %w", err)
	}

	// pq.QuoteIdentifier ile SQL injection koruması
	quotedSchema := pq.QuoteIdentifier(schemaName)
	setCmd := fmt.Sprintf("SET search_path TO %s, public", quotedSchema)

	if _, err := m.db.ExecContext(ctx, setCmd); err != nil {
		if m.logger != nil {
			m.logger.WithFields(logger.Fields{
				"schema_name": schemaName,
				"error":       err.Error(),
			}).Error("search_path ayarlanamadı")
		}
		return fmt.Errorf("search_path ayarlanamadı: %w", err)
	}

	if m.logger != nil {
		m.logger.WithFields(logger.Fields{
			"schema_name": schemaName,
		}).Debug("search_path ayarlandı")
	}

	return nil
}

// ResetSearchPath, search_path'i public schema'ya döndürür.
func (m *connectionManager) ResetSearchPath(ctx context.Context) error {
	resetCmd := "SET search_path TO public"

	if _, err := m.db.ExecContext(ctx, resetCmd); err != nil {
		if m.logger != nil {
			m.logger.WithFields(logger.Fields{
				"error": err.Error(),
			}).Error("search_path sıfırlanamadı")
		}
		return fmt.Errorf("search_path sıfırlanamadı: %w", err)
	}

	if m.logger != nil {
		m.logger.Debug("search_path public'e döndürüldü")
	}

	return nil
}

// ExecuteInTenantContext, verilen fonksiyonu tenant schema context'inde çalıştırır.
//
// ÖNEMLİ: sql.Conn kullanma sebebi:
// - sql.DB: Connection pool (birden fazla bağlantı)
// - sql.Conn: Pool'dan alınmış TEK bağlantı
// - search_path sql.Conn üzerinde set edilir, böylece diğer request'ler etkilenmez
// - Connection Close() ile pool'a geri döner (reusable)
func (m *connectionManager) ExecuteInTenantContext(ctx context.Context, schemaName string, fn func() error) error {
	if err := validateSchemaName(schemaName); err != nil {
		return fmt.Errorf("geçersiz schema adı: %w", err)
	}

	// Pool'dan dedicated connection al
	conn, err := m.db.Conn(ctx)
	if err != nil {
		if m.logger != nil {
			m.logger.WithFields(logger.Fields{
				"error": err.Error(),
			}).Error("Database bağlantısı alınamadı")
		}
		return fmt.Errorf("database bağlantısı alınamadı: %w", err)
	}

	// Connection'ı pool'a geri döndür (panic olsa bile)
	defer conn.Close()

	// Tenant schema'sını aktif et
	quotedSchema := pq.QuoteIdentifier(schemaName)
	setCmd := fmt.Sprintf("SET search_path TO %s, public", quotedSchema)

	if _, err := conn.ExecContext(ctx, setCmd); err != nil {
		if m.logger != nil {
			m.logger.WithFields(logger.Fields{
				"schema_name": schemaName,
				"error":       err.Error(),
			}).Error("Connection'da search_path ayarlanamadı")
		}
		return fmt.Errorf("search_path ayarlanamadı: %w", err)
	}

	// İşlem bitince search_path'i temizle (güvenlik kritik!)
	defer func() {
		// Context iptal olmuş olsa bile reset yapılmalı
		resetCtx := context.Background()

		if _, err := conn.ExecContext(resetCtx, "SET search_path TO public"); err != nil {
			if m.logger != nil {
				m.logger.WithFields(logger.Fields{
					"schema_name": schemaName,
					"error":       err.Error(),
				}).Error("defer'da search_path sıfırlanamadı")
			}
		}
	}()

	// Kullanıcı fonksiyonunu çalıştır
	if m.logger != nil {
		m.logger.WithFields(logger.Fields{
			"schema_name": schemaName,
		}).Debug("Tenant context'inde fonksiyon çalıştırılıyor")
	}

	return fn()
}

// GetConnection, pool'dan yeni bir bağlantı döndürür.
func (m *connectionManager) GetConnection(ctx context.Context) (*sql.Conn, error) {
	conn, err := m.db.Conn(ctx)
	if err != nil {
		if m.logger != nil {
			m.logger.WithFields(logger.Fields{
				"error": err.Error(),
			}).Error("Database bağlantısı alınamadı")
		}
		return nil, fmt.Errorf("database bağlantısı alınamadı: %w", err)
	}

	return conn, nil
}

// validateSchemaName, schema adının güvenli olduğunu doğrular.
//
// Kurallar:
// - 'tenant_' ile başlamalı
// - Sadece küçük harf, rakam ve underscore
// - Maksimum 63 karakter (PostgreSQL limiti)
//
// Sebep: SET search_path komutunda schema adı parameterize edilemiyor,
// bu yüzden SQL injection'a karşı manuel validasyon şart.
func validateSchemaName(schemaName string) error {
	if len(schemaName) == 0 || len(schemaName) > 63 {
		return fmt.Errorf("schema adı 1-63 karakter arasında olmalı")
	}

	// Tüm tenant schema'ları 'tenant_' ile başlar
	// Bu sayede system schema'larına (pg_catalog, information_schema) erişim engellenir
	if len(schemaName) < 7 || schemaName[:7] != "tenant_" {
		return fmt.Errorf("schema adı 'tenant_' ile başlamalı")
	}

	// Sadece güvenli karakterlere izin ver (SQL injection koruması)
	for i, ch := range schemaName {
		isLower := ch >= 'a' && ch <= 'z'
		isDigit := ch >= '0' && ch <= '9'
		isUnderscore := ch == '_'

		if !isLower && !isDigit && !isUnderscore {
			return fmt.Errorf("schema adında geçersiz karakter (pos %d): %c", i, ch)
		}
	}

	return nil
}
