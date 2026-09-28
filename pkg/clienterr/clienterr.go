// Package clienterr marks an error whose message is written for the client.
//
// Handlers used to answer every failure with err.Error(). Usecases wrap repository and
// provider errors with %w, so an outage reached the client as the driver's text, host and
// port included, and a payment provider's raw response body went the same way. A handler
// now shows a message only when the error chain holds an *Error; anything else is logged
// and answered with a generic 500.
package clienterr

// Error is an error whose message a transport may show the caller.
type Error struct{ msg string }

// New returns an error whose message is safe to show the caller. Declare it as a package
// variable and compare with errors.Is, as with errors.New.
func New(msg string) *Error { return &Error{msg: msg} }

// Error returns the message.
func (e *Error) Error() string { return e.msg }
