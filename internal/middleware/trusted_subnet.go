package middleware

import (
	"net"
	"net/http"
)

// TrustedSubnet allows requests whose X-Real-IP header belongs to subnet.
// A nil subnet disables the check.
func TrustedSubnet(subnet *net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if subnet == nil {
				next.ServeHTTP(w, r)
				return
			}
			ip := net.ParseIP(r.Header.Get("X-Real-IP"))
			if ip == nil || !subnet.Contains(ip) {
				http.Error(w, "agent IP is outside trusted subnet", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
