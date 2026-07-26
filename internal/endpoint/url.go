package endpoint

import (
	"net"
	"net/url"
)

func LocalhostAddress(port Port) string {
	return net.JoinHostPort("localhost", port.Decimal())
}

// LocalhostURL construye una URL sin aceptar un host controlado por el caller.
func LocalhostURL(port Port, path string, query url.Values) string {
	u := url.URL{
		Scheme:   "http",
		Host:     LocalhostAddress(port),
		Path:     path,
		RawQuery: query.Encode(),
	}
	return u.String()
}
