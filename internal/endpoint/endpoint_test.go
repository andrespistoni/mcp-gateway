package endpoint

import (
	"net/url"
	"strings"
	"testing"
)

func TestPortBoundaries(t *testing.T) {
	for _, value := range []int{MinPort, MaxPort} {
		port, err := NewPort(value)
		if err != nil || port.Number() != value {
			t.Fatalf("NewPort(%d) = %v, %v", value, port, err)
		}
	}
	for _, value := range []int{-1, 0, MinPort - 1, MaxPort + 1} {
		if _, err := NewPort(value); err == nil {
			t.Fatalf("NewPort(%d) debía fallar", value)
		}
	}
	for _, value := range []string{"", "+3333", " 3333", "3333x"} {
		if _, err := ParsePort(value); err == nil {
			t.Fatalf("ParsePort(%q) debía fallar", value)
		}
	}
}

func TestResolvePortPrecedenceAndPureDefaultAddress(t *testing.T) {
	configured := MustPort(4444)
	flag := MustPort(5555)
	if got := ResolvePort(&flag, &configured); got.Number() != 5555 {
		t.Fatalf("precedencia CLI = %d", got.Number())
	}
	if got := ResolvePort(nil, &configured); got.Number() != 4444 {
		t.Fatalf("precedencia config = %d", got.Number())
	}
	requested := ""
	binder := func(address string) { requested = address }
	binder(LocalhostAddress(ResolvePort(nil, nil)))
	if requested != "localhost:3333" {
		t.Fatalf("binder fake recibió %q", requested)
	}
}

func TestLocalhostURLNeverUsesIP(t *testing.T) {
	query := url.Values{"projectDir": {"/tmp/a b"}}
	value := LocalhostURL(MustPort(3333), "/sse", query)
	if value != "http://localhost:3333/sse?projectDir=%2Ftmp%2Fa+b" {
		t.Fatalf("URL = %q", value)
	}
	for _, forbidden := range []string{"127.0.0.1", "::1", "0.0.0.0"} {
		if strings.Contains(value, forbidden) {
			t.Fatalf("URL contiene IP prohibida %q", forbidden)
		}
	}
}
