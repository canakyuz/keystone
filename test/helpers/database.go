package helpers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	pq "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/pkg/tenantctx"
)

// TestDBConfig holds test database configuration
type TestDBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DefaultTestDBConfig, test veritabanı ayarlarını döner.
// Değerler ortam değişkenleriyle ezilebilir; CI'da Postgres servisi farklı
// host/port/parola ile geldiği için bu şart.
func DefaultTestDBConfig() *TestDBConfig {
	return &TestDBConfig{
		Host:     envOr("TEST_DB_HOST", "localhost"),
		Port:     envOr("TEST_DB_PORT", "5432"),
		User:     envOr("TEST_DB_USER", "postgres"),
		Password: envOr("TEST_DB_PASSWORD", "postgres"),
		DBName:   envOr("TEST_DB_NAME", "keystone_test"),
		SSLMode:  envOr("TEST_DB_SSLMODE", "disable"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// dbCounter, aynı süreç içindeki testlere çakışmayan veritabanı adı üretir.
var dbCounter atomic.Uint64

// uniqueDBName, her test için ayrı bir veritabanı adı üretir.
//
// NEDEN: önceden tüm testler tek bir sabit adı (keystone_test) DROP/CREATE
// ediyordu. `go test ./...` paketleri paralel koştuğu için iki test aynı anda
// aynı veritabanını silmeye çalışıyor ve "database is being accessed by other
// users" hatası alıyordu. İzole ad, çakışmayı tasarımdan kaldırır.
//
// Ad 63 bayt Postgres sınırına kırpılır.
func uniqueDBName(t *testing.T, base string) string {
	t.Helper()

	sanitized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + 32
		default:
			return '_'
		}
	}, t.Name())

	name := fmt.Sprintf("%s_%s_%d_%d", base, sanitized, os.Getpid(), dbCounter.Add(1))
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}

// dropDatabase, veritabanını siler. Silmeden önce artakalan bağlantıları
// sonlandırır: sql.DB havuzu Close() sonrası bağlantıyı hemen bırakmayabilir
// ve DROP DATABASE tek bir açık bağlantıda bile başarısız olur.
func dropDatabase(db *sql.DB, name string) {
	quoted := pq.QuoteIdentifier(name)

	_, _ = db.Exec(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()`, name)

	_, _ = db.Exec("DROP DATABASE IF EXISTS " + quoted)
}

// SetupTestDB, teste özel izole bir veritabanı açar ve gerçek migration'ları koşar.
// Test bitiminde veritabanı düşürülür. Postgres erişilemiyorsa test atlanır.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	cfg := DefaultTestDBConfig()
	dbName := uniqueDBName(t, cfg.DBName)

	adminConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode,
	)

	adminDB := openDBOrSkip(t, adminConnStr)
	if adminDB == nil {
		return nil
	}
	defer adminDB.Close()

	dropDatabase(adminDB, dbName)

	if _, err := adminDB.Exec("CREATE DATABASE " + pq.QuoteIdentifier(dbName)); skipIfConnectionError(t, err) {
		return nil
	} else {
		require.NoErrorf(t, err, "test veritabanı oluşturulamadı: %s", dbName)
	}

	testConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, dbName, cfg.SSLMode,
	)

	testDB := openDBOrSkip(t, testConnStr)
	if testDB == nil {
		return nil
	}

	runMigrations(t, testDB)

	t.Cleanup(func() {
		testDB.Close()

		admin := openDBOrSkip(t, adminConnStr)
		if admin != nil {
			defer admin.Close()
			dropDatabase(admin, dbName)
		}
	})

	return testDB
}

// migrationsDir, bu dosyanın konumundan repo kökündeki migrations dizinini bulur.
// runtime.Caller kullanılır çünkü `go test` her paketi kendi dizininde çalıştırır;
// çalışma dizinine göreli bir yol paketten pakete değişirdi.
func migrationsDir(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "çağıran dosya konumu alınamadı")

	// test/helpers/database.go -> repo kökü
	root := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	dir := filepath.Join(root, "migrations")

	info, err := os.Stat(dir)
	require.NoErrorf(t, err, "migrations dizini bulunamadı: %s", dir)
	require.Truef(t, info.IsDir(), "%s bir dizin değil", dir)

	return dir
}

// runMigrations, gerçek migrations/*.up.sql dosyalarını sırayla çalıştırır.
//
// NEDEN dosyadan okunuyor: şema daha önce bu helper'ın içinde elle kopyalanıyordu ve
// zamanla production migration'larından saptı. Örneğin tenants.phone kolonu
// migration'da vardı ama kopyada yoktu; testler gerçekte var olmayan bir şemaya
// karşı koştu ve repository sorguları "column does not exist" ile patladı.
// Tek doğruluk kaynağı migrations/ dizinidir. İkinci bir kopya tutmak bu sapmayı
// kaçınılmaz kılar, dosyadan okumak ise yapısal olarak imkansız hale getirir.
//
// Karmaşıklık: O(m) dosya okuma + O(m) sıralı exec, m = migration sayısı.
func runMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(migrationsDir(t), "*.up.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "migrations/*.up.sql bulunamadı")

	// Glob sırayı garanti etmez; 001..026 sırası şema bağımlılıkları için zorunlu.
	sort.Strings(files)

	for _, file := range files {
		stmt, readErr := os.ReadFile(file)
		require.NoErrorf(t, readErr, "migration okunamadı: %s", file)

		if _, execErr := db.Exec(string(stmt)); skipIfConnectionError(t, execErr) {
			return
		} else {
			require.NoErrorf(t, execErr, "migration başarısız: %s", filepath.Base(file))
		}
	}
}
func SetTenantContext(ctx context.Context, db *sql.DB, tenantID string) error {
	if _, err := db.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
		return err
	}

	schemaName, err := fetchTenantSchemaName(ctx, db, tenantID)
	if err != nil {
		return err
	}

	stmt := fmt.Sprintf("SET search_path TO %s, public", pq.QuoteIdentifier(schemaName))
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return err
	}

	return nil
}

// ClearTenantContext clears the tenant context
func ClearTenantContext(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "SELECT set_config('app.current_tenant', '', true)"); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, "RESET search_path")
	return err
}

// WithTenantContext executes a function within a tenant context
func WithTenantContext(ctx context.Context, db *sql.DB, tenantID string, fn func() error) error {
	if err := SetTenantContext(ctx, db, tenantID); err != nil {
		return err
	}
	defer ClearTenantContext(ctx, db)

	return fn()
}

func openDBOrSkip(t *testing.T, connStr string) *sql.DB {
	db, err := sql.Open("postgres", connStr)
	if skipIfConnectionError(t, err) {
		return nil
	}
	require.NoError(t, err)

	if err := db.Ping(); err != nil {
		if isConnectionError(err) {
			db.Close()
			t.Skipf("Skipping database-backed test: %v", err)
			return nil
		}
		require.NoError(t, err)
	}

	return db
}

func skipIfConnectionError(t *testing.T, err error) bool {
	if err == nil {
		return false
	}

	if isConnectionError(err) {
		t.Skipf("Skipping database-backed test: %v", err)
		return true
	}

	return false
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	msg := err.Error()
	return strings.Contains(msg, "connect: operation not permitted") ||
		strings.Contains(msg, "connect: permission denied") ||
		strings.Contains(msg, "connect: connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "could not connect to server")
}

func fetchTenantSchemaName(ctx context.Context, db *sql.DB, tenantID string) (string, error) {
	var schemaName string
	if err := db.QueryRowContext(ctx, "SELECT schema_name FROM tenants WHERE id = $1", tenantID).Scan(&schemaName); err != nil {
		return "", err
	}
	return schemaName, nil
}

// SetupAppRoleDB, RLS'in gerçekten sınanabildiği bir bağlantı döner.
//
// NEDEN ayrı bir rol gerekiyor: PostgreSQL'de süper kullanıcılar Row Level
// Security policy'lerini her koşulda atlar; FORCE ROW LEVEL SECURITY bile bunu
// değiştirmez. Testler varsayılan olarak "postgres" süper kullanıcısıyla
// bağlandığı için, o bağlantı üzerinden yapılan bir izolasyon testi hiçbir şey
// kanıtlamaz: policy hiç devreye girmez ve test yanlış sebeple geçer ya da kalır.
//
// Tablolar bu role devredilir, çünkü production'da uygulama çoğunlukla şemayı
// oluşturan rolle bağlanır. Sahiplik senaryosunu birebir yeniden üretmek,
// migration 027'deki FORCE ayarının gerçekten iş görüp görmediğini sınar.
//
// Karmaşıklık: O(k) ALTER, k = public şemasındaki tablo sayısı.
func SetupAppRoleDB(t *testing.T, admin *sql.DB) *sql.DB {
	t.Helper()

	cfg := DefaultTestDBConfig()

	var dbName string
	require.NoError(t, admin.QueryRow("SELECT current_database()").Scan(&dbName))

	roleName := uniqueDBName(t, "app_role")
	quotedRole := pq.QuoteIdentifier(roleName)

	mustExec := func(query string) {
		_, err := admin.Exec(query)
		require.NoErrorf(t, err, "hazırlık başarısız: %s", query)
	}

	mustExec("DROP ROLE IF EXISTS " + quotedRole)
	mustExec(fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD 'approle'", quotedRole))
	mustExec(fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", pq.QuoteIdentifier(dbName), quotedRole))
	mustExec(fmt.Sprintf("GRANT USAGE, CREATE ON SCHEMA public TO %s", quotedRole))

	// Tüm public tabloları ve sequence'ları role devret.
	mustExec(fmt.Sprintf(`
		DO $$
		DECLARE r record;
		BEGIN
			FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'public' LOOP
				EXECUTE format('ALTER TABLE public.%%I OWNER TO %s', r.tablename);
			END LOOP;
			FOR r IN SELECT sequencename FROM pg_sequences WHERE schemaname = 'public' LOOP
				EXECUTE format('ALTER SEQUENCE public.%%I OWNER TO %s', r.sequencename);
			END LOOP;
		END $$`, quotedRole, quotedRole))

	appDB, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%s user=%s password=approle dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, roleName, dbName, cfg.SSLMode,
	))
	require.NoError(t, err)

	// SET oturum bazlıdır. Havuz birden fazla bağlantı açarsa tenant context'i
	// taşımayan bir bağlantıya düşülebilir, bu da testi anlamsız kılar.
	appDB.SetMaxOpenConns(1)

	require.NoError(t, appDB.Ping())

	t.Cleanup(func() {
		appDB.Close()
		_, _ = admin.Exec("REASSIGN OWNED BY " + quotedRole + " TO CURRENT_USER")
		_, _ = admin.Exec("DROP OWNED BY " + quotedRole)
		_, _ = admin.Exec("DROP ROLE IF EXISTS " + quotedRole)
	})

	return appDB
}

// WithTenantSchema, repository'lerin beklediği tenant schema'sını context'e koyar.
//
// NEDEN: TenantConnectionManager refactor'ından sonra repository metotları
// schema adını context'ten okuyor ve yoksa hata döndürüyor. Production'da bunu
// TenantContextMiddleware yerleştirir; testlerde HTTP katmanı olmadığı için
// aynı anahtarı doğrudan koymak gerekiyor.
func WithTenantSchema(ctx context.Context, schemaName string) context.Context {
	return tenantctx.WithSchema(ctx, schemaName)
}
