package registry

import (
	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/canakyuz/keystone/internal/domain/registry"
)

// rowScanner is what *sql.Row and *sql.Rows have in common, so one decoder serves a
// single lookup and a list.
type rowScanner interface {
	Scan(dest ...any) error
}

// moduleColumns selects a module row in the order scanModule reads it.
//
// The entity holds plain strings, numbers and booleans, and database/sql refuses to put a
// NULL into any of them, so every column the schema lets be NULL is coalesced here. That
// includes the ones with a default: a default applies only when an insert leaves the
// column out. JSONB and array columns are not coalesced; scanModule decodes those itself.
const moduleColumns = `
	id, name, slug, code, display_name, description,
	category, module_type, status, is_public, is_beta,
	version, COALESCE(min_platform_version, ''), pricing_model,
	COALESCE(base_price, 0), COALESCE(currency, ''), COALESCE(billing_cycle, ''),
	features, capabilities, COALESCE(icon, ''), COALESCE(cover_image, ''), screenshots,
	COALESCE(demo_url, ''), COALESCE(documentation_url, ''),
	COALESCE(requires_database, FALSE), COALESCE(requires_storage, FALSE), COALESCE(requires_email, FALSE),
	database_tables, default_limits, COALESCE(installation_notes, ''), configuration_schema,
	tags, metadata,
	COALESCE(install_count, 0), COALESCE(rating, 0), COALESCE(review_count, 0),
	created_at, updated_at, deleted_at, COALESCE(created_by::text, ''), COALESCE(updated_by::text, '')`

// toolColumns selects a tool row in the order scanTool reads it, on the rules of
// moduleColumns.
const toolColumns = `
	id, name, slug, code, display_name, description,
	category, tool_type, scope, status, is_public, is_beta,
	version, COALESCE(min_platform_version, ''), pricing_model,
	COALESCE(base_price, 0), COALESCE(currency, ''), COALESCE(billing_cycle, ''),
	COALESCE(transaction_fee_percentage, 0), COALESCE(transaction_fee_fixed, 0),
	features, capabilities, COALESCE(icon, ''), COALESCE(cover_image, ''), screenshots,
	COALESCE(demo_url, ''), COALESCE(documentation_url, ''),
	COALESCE(requires_api_keys, FALSE), COALESCE(requires_webhook, FALSE),
	COALESCE(requires_storage, FALSE), COALESCE(requires_database, FALSE), database_tables,
	COALESCE(integration_provider, ''), COALESCE(integration_type, ''), api_endpoints,
	default_limits, rate_limits, configuration_schema, default_configuration,
	tags, metadata,
	COALESCE(install_count, 0), COALESCE(rating, 0), COALESCE(review_count, 0),
	created_at, updated_at, deleted_at, COALESCE(created_by::text, ''), COALESCE(updated_by::text, '')`

// scanModule decodes one row selected with moduleColumns.
//
// JSONB goes through a []byte first, for two reasons. database/sql has no conversion from
// NULL to json.RawMessage, and the driver hands back a slice of its read buffer: a scan
// into *[]byte copies it, while the reflection path a named byte slice takes does not, so
// the value would change under the entity on the connection's next read. TEXT[] needs
// pq.Array, which also turns NULL into a nil slice.
func scanModule(row rowScanner) (*registry.Module, error) {
	m := &registry.Module{}
	var features, capabilities, limits, schema, metadata []byte

	err := row.Scan(
		&m.ID, &m.Name, &m.Slug, &m.Code, &m.DisplayName, &m.Description,
		&m.Category, &m.ModuleType, &m.Status, &m.IsPublic, &m.IsBeta,
		&m.Version, &m.MinPlatformVersion, &m.PricingModel,
		&m.BasePrice, &m.Currency, &m.BillingCycle,
		&features, &capabilities, &m.Icon, &m.CoverImage, pq.Array(&m.Screenshots),
		&m.DemoURL, &m.DocumentationURL,
		&m.RequiresDatabase, &m.RequiresStorage, &m.RequiresEmail,
		pq.Array(&m.DatabaseTables), &limits, &m.InstallationNotes, &schema,
		pq.Array(&m.Tags), &metadata,
		&m.InstallCount, &m.Rating, &m.ReviewCount,
		&m.CreatedAt, &m.UpdatedAt, &m.DeletedAt, &m.CreatedBy, &m.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}

	m.Features, m.Capabilities, m.DefaultLimits, m.ConfigurationSchema, m.Metadata =
		features, capabilities, limits, schema, metadata
	return m, nil
}

// scanTool decodes one row selected with toolColumns, on the rules of scanModule.
func scanTool(row rowScanner) (*registry.Tool, error) {
	t := &registry.Tool{}
	var features, capabilities, endpoints, limits, rateLimits, schema, defaults, metadata []byte

	err := row.Scan(
		&t.ID, &t.Name, &t.Slug, &t.Code, &t.DisplayName, &t.Description,
		&t.Category, &t.ToolType, &t.Scope, &t.Status, &t.IsPublic, &t.IsBeta,
		&t.Version, &t.MinPlatformVersion, &t.PricingModel,
		&t.BasePrice, &t.Currency, &t.BillingCycle,
		&t.TransactionFeePercentage, &t.TransactionFeeFixed,
		&features, &capabilities, &t.Icon, &t.CoverImage, pq.Array(&t.Screenshots),
		&t.DemoURL, &t.DocumentationURL,
		&t.RequiresAPIKeys, &t.RequiresWebhook,
		&t.RequiresStorage, &t.RequiresDatabase, pq.Array(&t.DatabaseTables),
		&t.IntegrationProvider, &t.IntegrationType, &endpoints,
		&limits, &rateLimits, &schema, &defaults,
		pq.Array(&t.Tags), &metadata,
		&t.InstallCount, &t.Rating, &t.ReviewCount,
		&t.CreatedAt, &t.UpdatedAt, &t.DeletedAt, &t.CreatedBy, &t.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}

	t.Features, t.Capabilities, t.APIEndpoints = features, capabilities, endpoints
	t.DefaultLimits, t.RateLimits, t.ConfigurationSchema, t.DefaultConfiguration, t.Metadata =
		limits, rateLimits, schema, defaults, metadata
	return t, nil
}

// catalogSortColumns maps the sort keys a catalogue request may name to the columns behind
// them.
//
// The key arrives in the query string and goes into ORDER BY, where a bind parameter
// cannot stand in for a column. Before this map the text went in as sent, and the database
// evaluated whatever expression a caller put there. Only a key found here reaches the SQL.
var catalogSortColumns = map[string]string{
	"name":          "name",
	"install_count": "install_count",
	"rating":        "rating",
	"created_at":    "created_at",
}

// catalogOrderBy builds the ORDER BY clause for a module or tool list. An unknown key sorts
// by the default rather than failing, as an empty one always has. The id comes last so rows
// that share a value keep one order from page to page.
func catalogOrderBy(sortBy, sortOrder string) string {
	column, ok := catalogSortColumns[sortBy]
	if !ok {
		column = "created_at"
	}

	direction := "DESC"
	if sortOrder == "asc" {
		direction = "ASC"
	}

	return " ORDER BY " + column + " " + direction + ", id"
}

// isUUID reports whether s is a UUID in the hyphenated form Postgres accepts. Asking the
// database about anything else fails with a syntax error rather than finding no row, and
// an id that cannot exist is simply not found.
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}
