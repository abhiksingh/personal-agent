package twilio

import (
	"io"
)

const twilioProviderResponseBodyLimitBytes = int64(1 * 1024 * 1024)

func readBoundedTwilioProviderResponseBody(reader io.Reader) ([]byte, bool, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, twilioProviderResponseBodyLimitBytes+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(payload)) > twilioProviderResponseBodyLimitBytes {
		return payload[:twilioProviderResponseBodyLimitBytes], true, nil
	}
	return payload, false, nil
}
