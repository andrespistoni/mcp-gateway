package endpoint

import (
	"encoding/json"
	"testing"
)

func TestPortParsingFormattingAndYAML(t *testing.T) {
	port, err := ParsePort("3333")
	if err != nil || port.Decimal() != "3333" || port.Number() != 3333 {
		t.Fatalf("ParsePort = %#v, %v", port, err)
	}
	if _, err := ParsePort("999999999999999999999"); err == nil {
		t.Fatal("overflow decimal debía fallar")
	}
	if _, err := MustPort(3333).MarshalYAML(); err != nil {
		t.Fatal(err)
	}
	if _, err := (Port{}).MarshalYAML(); err == nil {
		t.Fatal("Port cero no debía serializar")
	}
	if encoded, err := json.Marshal(port.Number()); err != nil || string(encoded) != "3333" {
		t.Fatalf("encoded = %s, %v", encoded, err)
	}
}
