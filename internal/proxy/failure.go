package proxy

import "errors"

func IsUnavailable(err error) bool {
	return errors.Is(err, ErrDownstreamUnavailable)
}
