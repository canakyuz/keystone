package authn

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secret = "test-secret"

// sign issues a token with the given claims and method.
func sign(t *testing.T, method jwt.SigningMethod, key any, claims *Claims) string {
	t.Helper()

	signed, err := jwt.NewWithClaims(method, claims).SignedString(key)
	require.NoError(t, err)

	return signed
}

// validClaims returns claims that are in date.
func validClaims(tenantID string) *Claims {
	return &Claims{
		UserID:   "00000000-0000-4000-8000-000000000001",
		TenantID: tenantID,
		Email:    "user@example.test",
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
}

// TestVerify_AcceptsAValidToken is the baseline.
func TestVerify_AcceptsAValidToken(t *testing.T) {
	token := sign(t, jwt.SigningMethodHS256, []byte(secret), validClaims("tenant-a"))

	claims, err := Verify("Bearer "+token, secret)

	require.NoError(t, err)
	assert.Equal(t, "tenant-a", claims.TenantID)
}

// TestVerify_ReturnsNilClaimsOnFailure is the important one.
//
// jwt.ParseWithClaims fills the claims struct from the token body before it checks the
// signature, and leaves it filled when the check fails. A caller reading the struct it
// passed in, rather than the value returned, would be reading fields an attacker chose.
// This asserts that a failure yields nothing to read.
func TestVerify_ReturnsNilClaimsOnFailure(t *testing.T) {
	forged := sign(t, jwt.SigningMethodHS256, []byte("wrong-secret"), validClaims("attacker"))

	claims, err := Verify("Bearer "+forged, secret)

	require.ErrorIs(t, err, ErrUnauthenticated)
	assert.Nil(t, claims, "claims were returned alongside an authentication failure")
}

// TestVerify_RejectsUnusableHeaders covers the shapes a transport can hand over.
func TestVerify_RejectsUnusableHeaders(t *testing.T) {
	token := sign(t, jwt.SigningMethodHS256, []byte(secret), validClaims("tenant-a"))

	for name, header := range map[string]string{
		"empty":        "",
		"no scheme":    token,
		"wrong scheme": "Basic " + token,
		"scheme only":  "Bearer",
		"empty token":  "Bearer ",
		"not a token":  "Bearer not-a-token",
	} {
		t.Run(name, func(t *testing.T) {
			claims, err := Verify(header, secret)

			assert.ErrorIs(t, err, ErrUnauthenticated)
			assert.Nil(t, claims)
		})
	}
}

// TestVerify_AcceptsAnyBearerCasing verifies the scheme comparison is case-insensitive.
//
// RFC 7235 makes the scheme case-insensitive, and clients do send "bearer". A transport
// splitting the header itself would be one more place to get this subtly wrong, which is
// why the whole header is passed in rather than a bare token.
func TestVerify_AcceptsAnyBearerCasing(t *testing.T) {
	token := sign(t, jwt.SigningMethodHS256, []byte(secret), validClaims("tenant-a"))

	for _, scheme := range []string{"Bearer", "bearer", "BEARER", "BeArEr"} {
		claims, err := Verify(scheme+" "+token, secret)

		require.NoErrorf(t, err, "scheme %q was rejected", scheme)
		assert.Equal(t, "tenant-a", claims.TenantID)
	}
}

// TestVerify_RejectsAnExpiredToken verifies expiry is enforced.
func TestVerify_RejectsAnExpiredToken(t *testing.T) {
	expired := validClaims("tenant-a")
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))

	claims, err := Verify("Bearer "+sign(t, jwt.SigningMethodHS256, []byte(secret), expired), secret)

	require.ErrorIs(t, err, ErrUnauthenticated)
	assert.Nil(t, claims)
}

// TestVerify_RejectsAnUnsignedToken confirms the pinned method holds.
//
// golang-jwt v5 already refuses "alg: none" on its own; this asserts the behaviour rather
// than assuming a dependency will keep it.
func TestVerify_RejectsAnUnsignedToken(t *testing.T) {
	unsigned := sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims("attacker"))

	claims, err := Verify("Bearer "+unsigned, secret)

	require.ErrorIs(t, err, ErrUnauthenticated)
	assert.Nil(t, claims)
}
