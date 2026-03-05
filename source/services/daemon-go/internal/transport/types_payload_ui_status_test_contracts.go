package transport

import (
	"encoding/json"
	"fmt"
	"strings"
)

// UIStatusTestOperationDetails captures typed externally-consumed test details with
// an extension map for future keys.
type UIStatusTestOperationDetails struct {
	PluginID              string         `json:"plugin_id,omitempty"`
	WorkerRegistered      *bool          `json:"worker_registered,omitempty"`
	WorkerState           string         `json:"worker_state,omitempty"`
	Configured            *bool          `json:"configured,omitempty"`
	CredentialsConfigured *bool          `json:"credentials_configured,omitempty"`
	Endpoint              string         `json:"endpoint,omitempty"`
	SMSNumber             string         `json:"sms_number,omitempty"`
	VoiceNumber           string         `json:"voice_number,omitempty"`
	ExecutePathReady      *bool          `json:"execute_path_ready,omitempty"`
	ExecutePathProbeCode  *int           `json:"execute_path_probe_status_code,omitempty"`
	ExecutePathProbeErr   string         `json:"execute_path_probe_error,omitempty"`
	Available             *bool          `json:"available,omitempty"`
	BinaryPath            string         `json:"binary_path,omitempty"`
	DryRun                *bool          `json:"dry_run,omitempty"`
	Stdout                string         `json:"stdout,omitempty"`
	Stderr                string         `json:"stderr,omitempty"`
	ProbeError            string         `json:"probe_error,omitempty"`
	Additional            map[string]any `json:"-"`
}

func (d UIStatusTestOperationDetails) IsZero() bool {
	return d.PluginID == "" &&
		d.WorkerRegistered == nil &&
		d.WorkerState == "" &&
		d.Configured == nil &&
		d.CredentialsConfigured == nil &&
		d.Endpoint == "" &&
		d.SMSNumber == "" &&
		d.VoiceNumber == "" &&
		d.ExecutePathReady == nil &&
		d.ExecutePathProbeCode == nil &&
		d.ExecutePathProbeErr == "" &&
		d.Available == nil &&
		d.BinaryPath == "" &&
		d.DryRun == nil &&
		d.Stdout == "" &&
		d.Stderr == "" &&
		d.ProbeError == "" &&
		len(d.Additional) == 0
}

func (d *UIStatusTestOperationDetails) Set(key string, value any) {
	if d == nil {
		return
	}
	normalized := strings.TrimSpace(key)
	switch normalized {
	case "plugin_id":
		d.PluginID = readAnyString(value)
	case "worker_registered":
		d.WorkerRegistered = readAnyBoolPointer(value)
	case "worker_state":
		d.WorkerState = readAnyString(value)
	case "configured":
		d.Configured = readAnyBoolPointer(value)
	case "credentials_configured":
		d.CredentialsConfigured = readAnyBoolPointer(value)
	case "endpoint":
		d.Endpoint = readAnyString(value)
	case "sms_number":
		d.SMSNumber = readAnyString(value)
	case "voice_number":
		d.VoiceNumber = readAnyString(value)
	case "execute_path_ready":
		d.ExecutePathReady = readAnyBoolPointer(value)
	case "execute_path_probe_status_code":
		d.ExecutePathProbeCode = readAnyIntPointer(value)
	case "execute_path_probe_error":
		d.ExecutePathProbeErr = readAnyString(value)
	case "available":
		d.Available = readAnyBoolPointer(value)
	case "binary_path":
		d.BinaryPath = readAnyString(value)
	case "dry_run":
		d.DryRun = readAnyBoolPointer(value)
	case "stdout":
		d.Stdout = readAnyString(value)
	case "stderr":
		d.Stderr = readAnyString(value)
	case "probe_error":
		d.ProbeError = readAnyString(value)
	default:
		if d.Additional == nil {
			d.Additional = map[string]any{}
		}
		d.Additional[normalized] = value
	}
}

func (d UIStatusTestOperationDetails) AsMap() map[string]any {
	result := cloneAnyMapShallow(d.Additional)
	setStringField(result, "plugin_id", d.PluginID)
	setBoolPointerField(result, "worker_registered", d.WorkerRegistered)
	setStringField(result, "worker_state", d.WorkerState)
	setBoolPointerField(result, "configured", d.Configured)
	setBoolPointerField(result, "credentials_configured", d.CredentialsConfigured)
	setStringField(result, "endpoint", d.Endpoint)
	setStringField(result, "sms_number", d.SMSNumber)
	setStringField(result, "voice_number", d.VoiceNumber)
	setBoolPointerField(result, "execute_path_ready", d.ExecutePathReady)
	setIntPointerField(result, "execute_path_probe_status_code", d.ExecutePathProbeCode)
	setStringField(result, "execute_path_probe_error", d.ExecutePathProbeErr)
	setBoolPointerField(result, "available", d.Available)
	setStringField(result, "binary_path", d.BinaryPath)
	setBoolPointerField(result, "dry_run", d.DryRun)
	setStringField(result, "stdout", d.Stdout)
	setStringField(result, "stderr", d.Stderr)
	setStringField(result, "probe_error", d.ProbeError)
	return result
}

func UIStatusTestOperationDetailsFromMap(value map[string]any) UIStatusTestOperationDetails {
	if len(value) == 0 {
		return UIStatusTestOperationDetails{}
	}
	result := UIStatusTestOperationDetails{}
	for key, item := range value {
		result.Set(key, item)
	}
	result.Additional = removeKnownKeys(value,
		"plugin_id",
		"worker_registered",
		"worker_state",
		"configured",
		"credentials_configured",
		"endpoint",
		"sms_number",
		"voice_number",
		"execute_path_ready",
		"execute_path_probe_status_code",
		"execute_path_probe_error",
		"available",
		"binary_path",
		"dry_run",
		"stdout",
		"stderr",
		"probe_error",
	)
	return result
}

func (d UIStatusTestOperationDetails) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.AsMap())
}

func (d *UIStatusTestOperationDetails) UnmarshalJSON(data []byte) error {
	if d == nil {
		return fmt.Errorf("nil UIStatusTestOperationDetails")
	}
	if len(strings.TrimSpace(string(data))) == 0 || string(data) == "null" {
		*d = UIStatusTestOperationDetails{}
		return nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*d = UIStatusTestOperationDetailsFromMap(decoded)
	return nil
}
