// Package outbound makes HTTP requests to addresses somebody else chose.
//
// Webhook delivery sends a request to a URL a tenant registered. That makes the tenant
// able to point this service at any address it can reach, and this service sits inside a
// network perimeter the tenant does not. A URL of http://169.254.169.254/ reaches the
// cloud metadata service and its credentials; http://10.0.0.5/admin reaches whatever is
// there and does not expect a request from inside; http://localhost:5432 reaches the
// database. That is server-side request forgery, and the request itself is the damage
// even when the response never comes back.
//
// Three defences, because each one alone has a hole:
//
//   - The URL is checked when it is registered. Necessary, and not sufficient: a hostname
//     that resolves to a public address today can resolve to 169.254.169.254 tomorrow.
//     That is DNS rebinding, and no amount of parsing catches it.
//   - The resolved address is checked when the connection is dialled. This is the one that
//     actually holds, because it inspects what the connection is really going to, at the
//     moment it goes there.
//   - Redirects are refused. A perfectly legitimate public URL can answer 302 with a
//     Location of anything at all, and a client that follows it has undone both checks.
package outbound

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrBlockedAddress reports a destination this service will not connect to.
var ErrBlockedAddress = fmt.Errorf("outbound: blocked address")

// blockedNets are the ranges a tenant-supplied URL must never reach.
//
// Listed explicitly rather than derived from net.IP helpers alone, because the helpers do
// not agree on every case that matters here: 0.0.0.0/8 and 100.64.0.0/10 are neither
// loopback nor "private" by their definition, and both route somewhere inside.
var blockedNets = func() []*net.IPNet {
	cidrs := []string{
		"0.0.0.0/8",      // this network; 0.0.0.0 reaches localhost on Linux
		"10.0.0.0/8",     // RFC 1918
		"100.64.0.0/10",  // carrier-grade NAT
		"127.0.0.0/8",    // loopback
		"169.254.0.0/16", // link-local, including the cloud metadata service
		"172.16.0.0/12",  // RFC 1918
		"192.0.0.0/24",   // IETF protocol assignments
		"192.168.0.0/16", // RFC 1918
		"198.18.0.0/15",  // benchmarking
		"224.0.0.0/4",    // multicast
		"240.0.0.0/4",    // reserved
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link-local
		"ff00::/8",       // IPv6 multicast
		"::/128",         // unspecified
		"64:ff9b::/96",   // IPv4/IPv6 translation, a way to spell an IPv4 target
	}

	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic("outbound: bad CIDR in the blocklist: " + cidr)
		}
		nets = append(nets, network)
	}

	return nets
}()

// IsBlocked says whether an address is one this service refuses to reach.
func IsBlocked(ip net.IP) bool {
	// An IPv4 address written as IPv6 is the same address. Without this, ::ffff:127.0.0.1
	// walks straight past an IPv4-only blocklist.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	for _, network := range blockedNets {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// ValidateURL checks a destination at the point it is registered.
//
// This is the cheap check that gives a caller a useful error message. It is not the
// security boundary — the dialler is — because a hostname's meaning can change between
// this call and the connection.
func ValidateURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("outbound: could not parse the url: %w", err)
	}

	// https only. Deliveries carry a signature, but the payload itself describes a
	// customer's tenant and sending it in clear text is a leak regardless of the header.
	if parsed.Scheme != "https" {
		return fmt.Errorf("%w: scheme %q is not allowed, use https", ErrBlockedAddress, parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("%w: the url has no host", ErrBlockedAddress)
	}

	// Credentials in the URL end up in logs and in the Host header's neighbourhood, and
	// there is no reason for a webhook destination to carry them.
	if parsed.User != nil {
		return fmt.Errorf("%w: the url must not contain credentials", ErrBlockedAddress)
	}

	host := parsed.Hostname()

	// A literal address can be checked now. A name cannot, and is left to the dialler.
	if ip := net.ParseIP(host); ip != nil && IsBlocked(ip) {
		return fmt.Errorf("%w: %s is not routable from here", ErrBlockedAddress, host)
	}

	return nil
}

// NewClient builds an HTTP client that refuses to reach an internal address.
//
// The guard is on the dialler, so it sees the address the connection actually resolved
// to. A check on the hostname would be defeated by a DNS record that answers with a
// public address once and a private one the next time.
func NewClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("outbound: could not split %q: %w", addr, err)
			}

			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("outbound: could not resolve %q: %w", host, err)
			}

			// Every answer is checked, not the first. A resolver that returns one public
			// address and one private one would otherwise be a way through, and which one
			// a dial picks is not something this code controls.
			for _, addr := range ips {
				if IsBlocked(addr.IP) {
					return nil, fmt.Errorf("%w: %s resolves to %s", ErrBlockedAddress, host, addr.IP)
				}
			}

			// Dialled by address rather than by name, so the connection goes to an address
			// that was checked. Re-resolving here would reopen the rebinding window between
			// the check above and the connection.
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			// Refused rather than followed-and-rechecked. A redirect is the receiver
			// choosing a second destination, and a webhook has no reason to need one.
			return http.ErrUseLastResponse
		},
	}
}
