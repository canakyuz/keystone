// Package authn verifies bearer tokens.
//
// It exists because there are now two transports. The HTTP middleware and the gRPC
// interceptor have to reach the same verdict about the same token, and the way to
// guarantee that is one implementation rather than two that look alike.
//
// Copying it would be worse than it sounds. The bug this repository already closed once
// was in exactly this area: the middleware read a context key the authenticator never
// wrote, the JWT path silently never ran, and every request fell through to an untrusted
// header. Two copies of that logic means two chances to make that mistake and only one
// of them gets fixed.
//
// Nothing here knows about HTTP or gRPC. The transports extract a string and interpret
// the error; this package decides whether the string is a valid token and what it says.
package authn

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ErrUnauthenticated is returned for any token that cannot be trusted.
//
// One error covers a missing header, a malformed one, a bad signature and an expired
// token on purpose. The caller cannot act differently on the difference, and telling an
// unauthenticated client which of the four it hit is free reconnaissance.
var ErrUnauthenticated = errors.New("authn: unauthenticated")

// Claims is the verified content of a token.
type Claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Verify parses and validates an Authorization header value.
//
// It takes the whole header rather than a bare token so that the "Bearer " handling is
// also shared: a transport that split it itself would be one more place to get the
// scheme comparison subtly wrong.
func Verify(authorization, secret string) (*Claims, error) {
	token, err := bearerToken(authorization)
	if err != nil {
		return nil, err
	}

	claims := &Claims{}

	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		// The signing method is pinned rather than assumed.
		//
		// This is defence in depth, not a hole being closed: golang-jwt v5 already refuses
		// "alg: none" on its own, and an RS256 token fails because this key is not an RSA
		// key. What the check buys is that the accepted algorithm is stated here instead of
		// inherited from a dependency's defaults, so widening it becomes a visible edit
		// rather than a side effect of an upgrade.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %v", ErrUnauthenticated, t.Header["alg"])
		}

		return []byte(secret), nil
	})
	// Both conditions matter, and the second is the one that is easy to drop.
	//
	// ParseWithClaims populates the claims struct from the token body *before* it
	// verifies the signature, and leaves it populated when verification fails. A caller
	// that checks only err, or that reads the struct it passed in rather than the value
	// returned here, is reading attacker-controlled fields. Returning nil on any failure
	// makes that mistake impossible from outside this package.
	if err != nil || !parsed.Valid {
		return nil, ErrUnauthenticated
	}

	return claims, nil
}

// bearerToken pulls the token out of an Authorization header value.
func bearerToken(authorization string) (string, error) {
	if authorization == "" {
		return "", ErrUnauthenticated
	}

	scheme, token, found := strings.Cut(authorization, " ")
	if !found || !strings.EqualFold(scheme, "bearer") || token == "" {
		return "", ErrUnauthenticated
	}

	return token, nil
}
