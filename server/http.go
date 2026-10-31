package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

func writeJSON(w http.ResponseWriter, status int, obj any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(obj)
}
func fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		fail(w, 400, "请求 JSON 无效")
		return false
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		fail(w, 400, "请求必须只含一个 JSON 对象")
		return false
	}
	return true
}
func secret(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func hashToken(t string) string { v := sha256.Sum256([]byte(t)); return hex.EncodeToString(v[:]) }
func cleanText(v string, max int) string {
	v = strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, v))
	if utf8.RuneCountInString(v) > max {
		return string([]rune(v)[:max])
	}
	return v
}
func pager(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}
	if page > 1000 {
		page = 1000
	}
	return limit, (page - 1) * limit
}

var trustedProxies []*net.IPNet

// Only allow a reverse proxy explicitly configured at startup to provide the
// caller IP. X-Real-IP MUST be overwritten by the proxy from its peer address;
// never trust an arbitrary X-Forwarded-For value from an internet client.
func configureTrustedProxies(raw string) error {
	trustedProxies = nil
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		_, network, err := net.ParseCIDR(part)
		if err != nil {
			return err
		}
		ones, bits := network.Mask.Size()
		if ones == 0 || bits == 0 {
			return errors.New("TRUSTED_PROXY_CIDRS must not trust the entire internet")
		}
		trustedProxies = append(trustedProxies, network)
	}
	return nil
}
func clientIP(r *http.Request) string {
	h, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		h = r.RemoteAddr
	}
	peer := net.ParseIP(h)
	for _, cidr := range trustedProxies {
		if peer != nil && cidr.Contains(peer) {
			if real := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); real != nil {
				return real.String()
			}
			break
		}
	}
	return h
}

type ipBucket struct {
	count int
	reset time.Time
}
type IPLimiter struct {
	mu      sync.Mutex
	entries map[string]ipBucket
}

func NewIPLimiter() *IPLimiter { return &IPLimiter{entries: make(map[string]ipBucket)} }
func (l *IPLimiter) allow(key string, limit int, period time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	v := l.entries[key]
	if now.After(v.reset) {
		v = ipBucket{reset: now.Add(period)}
	}
	v.count++
	l.entries[key] = v
	if len(l.entries) > 10000 {
		for k, b := range l.entries {
			if now.After(b.reset) {
				delete(l.entries, k)
			}
		}
	}
	return v.count <= limit
}
