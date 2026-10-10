package httpserver

import (
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func allowSecretSettings(w http.ResponseWriter, r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		httpx.Error(w, http.StatusForbidden, "secret settings are local only")
		return false
	}
	requestHost := r.Host
	if host, _, err := net.SplitHostPort(requestHost); err == nil {
		requestHost = host
	}
	requestHost = strings.Trim(requestHost, "[]")
	ip := net.ParseIP(requestHost)
	if !strings.EqualFold(requestHost, "localhost") && (ip == nil || !ip.IsLoopback()) {
		httpx.Error(w, http.StatusForbidden, "local hostname required")
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") {
			httpx.Error(w, http.StatusForbidden, "same origin required")
			return false
		}
	}
	return true
}
