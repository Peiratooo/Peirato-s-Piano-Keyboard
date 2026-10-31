package main

import (
	"net/http/httptest"
	"testing"
)

func TestTrustedProxyRequiresExplicitCIDR(t *testing.T) {
	if err := configureTrustedProxies("127.0.0.1/32"); err != nil {
		t.Fatal(err)
	}
	defer configureTrustedProxies("")
	direct := httptest.NewRequest("GET", "/", nil)
	direct.RemoteAddr = "203.0.113.9:34567"
	direct.Header.Set("X-Real-IP", "198.51.100.10")
	if ip := clientIP(direct); ip != "203.0.113.9" {
		t.Fatalf("spoofed header trusted: %s", ip)
	}
	viaProxy := httptest.NewRequest("GET", "/", nil)
	viaProxy.RemoteAddr = "127.0.0.1:34567"
	viaProxy.Header.Set("X-Real-IP", "198.51.100.10")
	if ip := clientIP(viaProxy); ip != "198.51.100.10" {
		t.Fatalf("trusted proxy ignored: %s", ip)
	}
	if err := configureTrustedProxies("0.0.0.0/0"); err == nil {
		t.Fatal("unsafe trust-all CIDR accepted")
	}
}
