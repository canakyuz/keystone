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
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/canakyuz/keystone/pkg/clienterr"
	"github.com/canakyuz/keystone/pkg/validator"
)

// Error answers err with status when it is an error written for the caller: a
// clienterr.Error, or a validation failure. Anything else is logged and answered with a
// generic 500.
//
// It writes the 500 itself rather than returning err to the application's error handler.
// Fiber's default handler writes err.Error() to the client, so returning err would make
// the guarantee depend on how the app that mounts the handler was configured.
func Error(c *fiber.Ctx, status int, err error) error {
	if msg, ok := clientMessage(err); ok {
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}

	log.Printf("unhandled error: %s %s: %v", c.Method(), c.Path(), err)

	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
}

// clientMessage returns the message of the client or validation error in err's chain. It
// is that error's own message, not err's, so a wrapper that added detail cannot leak it.
func clientMessage(err error) (string, bool) {
	var shown *clienterr.Error
	if errors.As(err, &shown) {
		return shown.Error(), true
	}

	var invalid validator.ValidationErrors
	if errors.As(err, &invalid) {
		return invalid.Error(), true
	}

	return "", false
}
