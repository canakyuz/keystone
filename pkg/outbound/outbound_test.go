package outbound

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsBlocked_CoversTheAddressesThatMatter names the ranges explicitly.
//
// Written as a table of real addresses rather than as a restatement of the CIDR list,
// because the failure worth catching is a range somebody forgot, and a test derived from
// the same list forgets it too.
func TestIsBlocked_CoversTheAddressesThatMatter(t *testing.T) {
	blocked := map[string]string{
		"169.254.169.254":        "the cloud metadata service, the classic SSRF target",
		"127.0.0.1":              "loopback",
		"0.0.0.0":                "reaches localhost on Linux",
		"10.1.2.3":               "RFC 1918",
		"172.16.0.1":             "RFC 1918",
		"192.168.1.1":            "RFC 1918",
		"100.64.0.1":             "carrier-grade NAT",
		"::1":                    "IPv6 loopback",
		"fe80::1":                "IPv6 link-local",
		"fd00::1":                "IPv6 unique local",
		"::ffff:127.0.0.1":       "loopback written as IPv6, which an IPv4-only check would miss",
		"::ffff:169.254.169.254": "the metadata service written as IPv6",
	}

	for address, why := range blocked {
		ip := net.ParseIP(address)
		require.NotNilf(t, ip, "%s is not a valid address", address)
		assert.Truef(t, IsBlocked(ip), "%s should be blocked: %s", address, why)
	}

	allowed := []string{"1.1.1.1", "8.8.8.8", "93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"}
	for _, address := range allowed {
		ip := net.ParseIP(address)
		require.NotNil(t, ip)
		assert.Falsef(t, IsBlocked(ip), "%s should be reachable", address)
	}
}

// TestValidateURL_RejectsWhatItShould covers the registration-time check.
func TestValidateURL_RejectsWhatItShould(t *testing.T) {
	rejected := map[string]string{
		"http://example.com/hook":            "plain http would send tenant data in clear",
		"https://169.254.169.254/latest":     "the metadata service by address",
		"https://127.0.0.1/hook":             "loopback",
		"https://[::1]/hook":                 "IPv6 loopback",
		"https://10.0.0.1/hook":              "a private address",
		"https://user:pass@example.com/hook": "credentials in the url",
		"ftp://example.com/hook":             "not a scheme this makes requests with",
		"file:///etc/passwd":                 "not a scheme at all",
		"https://":                           "no host",
		"":                                   "empty",
	}

	for raw, why := range rejected {
		assert.Errorf(t, ValidateURL(raw), "%q should be rejected: %s", raw, why)
	}

	for _, raw := range []string{
		"https://example.com/hook",
		"https://hooks.example.com:8443/keystone",
	} {
		assert.NoErrorf(t, ValidateURL(raw), "%q should be accepted", raw)
	}
}

// TestNewClient_RefusesToConnectToAnInternalAddress is the check that actually holds.
//
// Registration-time validation cannot survive DNS rebinding: a name that resolved to a
// public address when it was registered can resolve to 169.254.169.254 by the time the
// delivery goes out. The dialler inspects the address the connection is really about to
// use, which is the only place the question can be answered honestly.
func TestNewClient_RefusesToConnectToAnInternalAddress(t *testing.T) {
	// A real local server. Its address is loopback, which is exactly what a rebound DNS
	// record would point at.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(5 * time.Second)

	resp, err := client.Get(server.URL)
	if resp != nil {
		_ = resp.Body.Close()
	}

	require.Error(t, err, "the client connected to a loopback address")
	assert.Contains(t, err.Error(), "blocked address")
}

// TestNewClient_DoesNotFollowRedirects verifies the third defence.
//
// A public URL that answers 302 with a Location of http://169.254.169.254/ would undo
// both other checks, because the second request is made to an address nobody validated.
func TestNewClient_DoesNotFollowRedirects(t *testing.T) {
	var reachedTarget bool

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reachedTarget = true
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	// The guard is bypassed for this test so the redirect itself is what is under test,
	// not the loopback block.
	client := &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: NewClient(time.Second).CheckRedirect,
	}

	resp, err := client.Get(redirector.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusFound, resp.StatusCode, "the redirect was followed")
	assert.False(t, reachedTarget, "the client followed a redirect to a second destination")
}
