package tenant

import "fmt"

// This file gathers the conversion between the domain entity's optional fields and
// Postgres's nullable columns in one place.
//
// WHY it is needed: tenant.Tenant holds Phone / CustomDomain / CreatedBy / UpdatedBy
// fields as plain strings. On the Postgres side those columns are nullable, and two
// of them are UUIDs. Passing a plain string straight through caused two bugs:
//
//   - On write: an empty string reached a UUID column and produced
//     `invalid input syntax for type uuid: ""`.
//   - On read: a NULL column was scanned into a *string target and produced
//     `converting NULL to string is unsupported`.
//
// The alternative was making the entity fields *string. That was rejected: shaping
// the domain layer around database nullability inverts the layer dependency. The
// conversion belongs at the repository boundary.

// nullable turns an empty string into SQL NULL and passes a non-empty value through
// unchanged. Complexity: O(1).
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullString is a NULL-tolerant sql.Scanner target. On NULL it sets the target to
// the empty string, preserving the entity's representation of "no value".
type nullString struct {
	dst *string
}

// Scan implements sql.Scanner.
// Complexity: O(n), where n is the byte length of the value.
func (n nullString) Scan(src any) error {
	if src == nil {
		*n.dst = ""
		return nil
	}

	switch v := src.(type) {
	case string:
		*n.dst = v
	case []byte:
		*n.dst = string(v)
	default:
		return fmt.Errorf("nullString: desteklenmeyen kaynak tipi %T", src)
	}

	return nil
}
