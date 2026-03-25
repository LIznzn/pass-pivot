package handler

import (
	"errors"
	"io"
)

const maxUpstreamResponseBodyBytes int64 = 10 * 1024 * 1024

var errUpstreamResponseTooLarge = errors.New("upstream response body exceeds size limit")

func readUpstreamResponseBody(body io.Reader) ([]byte, error) {
	limited, err := io.ReadAll(io.LimitReader(body, maxUpstreamResponseBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(limited)) > maxUpstreamResponseBodyBytes {
		return nil, errUpstreamResponseTooLarge
	}
	return limited, nil
}
