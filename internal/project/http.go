package project

import (
	"fmt"
	"net/url"
	"unicode/utf8"
)

// FromHTTP recibe el valor raw percent-encoded de query y el valor UTF-8 del
// header. Los punteros distinguen ausencia de un valor presente pero vacío.
func FromHTTP(query, header *string) (OptionalDir, error) {
	var queryDir, headerDir Dir
	var err error
	if query != nil {
		decoded, decodeErr := url.QueryUnescape(*query)
		if decodeErr != nil || !utf8.ValidString(decoded) {
			return OptionalDir{}, fmt.Errorf("projectDir de query tiene codificación inválida")
		}
		queryDir, err = Resolve(decoded)
		if err != nil {
			return OptionalDir{}, fmt.Errorf("projectDir de query inválido: %w", err)
		}
	}
	if header != nil {
		if !utf8.ValidString(*header) {
			return OptionalDir{}, fmt.Errorf("X-Project-Dir no es UTF-8 válido")
		}
		headerDir, err = Resolve(*header)
		if err != nil {
			return OptionalDir{}, fmt.Errorf("X-Project-Dir inválido: %w", err)
		}
	}
	if query != nil && header != nil && !Compare(queryDir, headerDir) {
		return OptionalDir{}, fmt.Errorf("projectDir de query y header no coinciden")
	}
	if query != nil {
		return Some(queryDir), nil
	}
	if header != nil {
		return Some(headerDir), nil
	}
	return OptionalDir{}, nil
}
