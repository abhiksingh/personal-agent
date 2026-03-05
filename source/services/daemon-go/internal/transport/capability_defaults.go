package transport

var defaultCapabilitySmokeChannels = []string{
	"app",
	"message",
	"voice",
}

var defaultCapabilitySmokeConnectors = []string{
	"builtin.app",
	"imessage",
	"twilio",
	"apple.mail",
	"apple.calendar",
	"apple.browser",
	"apple.finder",
	"cloudflared",
}

func DefaultCapabilitySmokeChannels() []string {
	return append([]string{}, defaultCapabilitySmokeChannels...)
}

func DefaultCapabilitySmokeConnectors() []string {
	return append([]string{}, defaultCapabilitySmokeConnectors...)
}
