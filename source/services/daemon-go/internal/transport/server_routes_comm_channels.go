package transport

import "net/http"

func (s *Server) registerCommRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/comm/send", s.handleCommSend)
	mux.HandleFunc("/v1/comm/attempts", s.handleCommAttempts)
	mux.HandleFunc("/v1/comm/webhook-receipts/list", s.handleCommWebhookReceiptList)
	mux.HandleFunc("/v1/comm/ingest-receipts/list", s.handleCommIngestReceiptList)
	mux.HandleFunc("/v1/comm/policy/set", s.handleCommPolicySet)
	mux.HandleFunc("/v1/comm/policy/list", s.handleCommPolicyList)
	mux.HandleFunc("/v1/comm/messages/ingest", s.handleCommMessagesIngest)
	mux.HandleFunc("/v1/comm/mail/ingest", s.handleCommMailIngest)
	mux.HandleFunc("/v1/comm/calendar/ingest", s.handleCommCalendarIngest)
	mux.HandleFunc("/v1/comm/browser/ingest", s.handleCommBrowserIngest)
}

func (s *Server) registerTwilioChannelRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/channels/twilio/set", s.handleTwilioSet)
	mux.HandleFunc("/v1/channels/twilio/get", s.handleTwilioGet)
	mux.HandleFunc("/v1/channels/twilio/check", s.handleTwilioCheck)
	mux.HandleFunc("/v1/channels/twilio/sms-chat-turn", s.handleTwilioSMSChatTurn)
	mux.HandleFunc("/v1/channels/twilio/start-call", s.handleTwilioStartCall)
	mux.HandleFunc("/v1/channels/twilio/call-status", s.handleTwilioCallStatus)
	mux.HandleFunc("/v1/channels/twilio/transcript", s.handleTwilioTranscript)
	mux.HandleFunc("/v1/channels/twilio/ingest-sms", s.handleTwilioIngestSMS)
	mux.HandleFunc("/v1/channels/twilio/ingest-voice", s.handleTwilioIngestVoice)
	mux.HandleFunc("/v1/channels/twilio/webhook/serve", s.handleTwilioWebhookServe)
	mux.HandleFunc("/v1/channels/twilio/webhook/replay", s.handleTwilioWebhookReplay)
	mux.HandleFunc("/v1/channels/mappings/list", s.handleChannelConnectorMappingList)
	mux.HandleFunc("/v1/channels/mappings/upsert", s.handleChannelConnectorMappingUpsert)
}

func (s *Server) registerCloudflaredConnectorRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/connectors/cloudflared/version", s.handleCloudflaredVersion)
	mux.HandleFunc("/v1/connectors/cloudflared/exec", s.handleCloudflaredExec)
}
