package daemonruntime

import (
	"io"
)

const daemonWorkerRPCResponseBodyLimitBytes = int64(1 * 1024 * 1024)

func readBoundedHTTPResponseBody(reader io.Reader, maxBytes int64) ([]byte, bool, error) {
	if maxBytes <= 0 {
		body, err := io.ReadAll(reader)
		if err != nil {
			return nil, false, err
		}
		return body, false, nil
	}
	body, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(body)) > maxBytes {
		return body[:maxBytes], true, nil
	}
	return body, false, nil
}
