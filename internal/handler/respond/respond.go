// Package respond writes a usecase error to the client without leaking what the client
// should not see.
//
// The handlers used to write err.Error() for every failure. The usecases wrap repository
// errors with %w, so a database failure reached the client as the driver's text, and it did
// so under whatever status the handler had picked for the expected failure: a lookup that
// failed because the database was down answered 404.
package respond

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/canakyuz/keystone/internal/domain/tenant"
	"github.com/canakyuz/keystone/internal/domain/user"
	"github.com/canakyuz/keystone/pkg/validator"
)

// Error answers err with status when it is an error written for the caller: a tenant or
// user domain error, or a validation failure. Anything else is returned unchanged, and the
// application's error handler logs it and answers a generic 500.
func Error(c *fiber.Ctx, status int, err error) error {
	if msg, ok := clientMessage(err); ok {
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}

	return err
}

// clientMessage returns the message of the domain or validation error in err's chain. It
// is that error's own message, not err's, so a wrapper that added detail cannot leak it.
func clientMessage(err error) (string, bool) {
	var tenantErr *tenant.Error
	if errors.As(err, &tenantErr) {
		return tenantErr.Error(), true
	}

	var userErr *user.Error
	if errors.As(err, &userErr) {
		return userErr.Error(), true
	}

	var invalid validator.ValidationErrors
	if errors.As(err, &invalid) {
		return invalid.Error(), true
	}

	return "", false
}
