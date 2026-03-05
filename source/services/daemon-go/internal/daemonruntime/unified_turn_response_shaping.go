package daemonruntime

import (
	"strings"

	"personalagent/runtime/internal/transport"
)

type channelResponseProfile struct {
	ProfileID          string
	StyleDirective     string
	PromptInstructions []string
	ProfileGuardrails  []string
}

type resolvedResponseShapingPolicy struct {
	ChannelID           string
	ProfileID           string
	PersonaSource       string
	StylePrompt         string
	Guardrails          []string
	ChannelInstructions []string
}

var channelResponseProfiles = map[string]channelResponseProfile{
	"app": {
		ProfileID:      "app.default",
		StyleDirective: "Optimize for in-app reading with concise but complete response structure.",
		PromptInstructions: []string{
			"Use short, structured paragraphs suitable for desktop reading.",
			"Include concrete next steps when action is blocked or pending approval.",
		},
		ProfileGuardrails: []string{
			"Do not rely on markdown-only affordances to convey required actions.",
		},
	},
	"message": {
		ProfileID:      "message.compact",
		StyleDirective: "Optimize for short async messaging where the user scans quickly.",
		PromptInstructions: []string{
			"Keep replies compact and front-load the most important action outcome.",
			"Prefer one clear next action when follow-up is required.",
		},
		ProfileGuardrails: []string{
			"Avoid verbose explanations unless the user explicitly asks for detail.",
		},
	},
	"voice": {
		ProfileID:      "voice.spoken",
		StyleDirective: "Optimize for spoken delivery and text-to-speech clarity.",
		PromptInstructions: []string{
			"Use natural spoken sentences with clear pacing and simple punctuation.",
			"Avoid dense formatting, code-like shorthand, and ambiguous abbreviations.",
		},
		ProfileGuardrails: []string{
			"Keep sentence structure easy to read aloud without losing required action details.",
		},
	},
}

func resolveResponseShapingPolicy(persona transport.ChatPersonaPolicyResponse, channelID string) resolvedResponseShapingPolicy {
	normalizedChannel := normalizeTurnChannelID(channelID)
	profile := channelResponseProfileForChannel(normalizedChannel)

	stylePrompt := strings.TrimSpace(persona.StylePrompt)
	if stylePrompt == "" {
		stylePrompt = defaultChatPersonaStylePrompt
	}
	if directive := strings.TrimSpace(profile.StyleDirective); directive != "" {
		stylePrompt = strings.TrimSpace(stylePrompt + "\n\nChannel profile directive: " + directive)
	}

	guardrails := normalizeGuardrails(append(append([]string{}, persona.Guardrails...), profile.ProfileGuardrails...))
	if len(guardrails) == 0 {
		guardrails = append([]string(nil), defaultChatPersonaGuardrails...)
	}

	personaSource := strings.TrimSpace(persona.Source)
	if personaSource == "" {
		personaSource = "default"
	}

	return resolvedResponseShapingPolicy{
		ChannelID:           normalizedChannel,
		ProfileID:           strings.TrimSpace(profile.ProfileID),
		PersonaSource:       personaSource,
		StylePrompt:         stylePrompt,
		Guardrails:          guardrails,
		ChannelInstructions: append([]string(nil), profile.PromptInstructions...),
	}
}

func withResponseShapingMetadata(metadata map[string]any, policy resolvedResponseShapingPolicy) map[string]any {
	resolved := map[string]any{}
	for key, value := range metadata {
		resolved[key] = value
	}
	resolved["response_shaping_channel"] = strings.TrimSpace(policy.ChannelID)
	resolved["response_shaping_profile"] = strings.TrimSpace(policy.ProfileID)
	resolved["persona_policy_source"] = strings.TrimSpace(policy.PersonaSource)
	resolved["response_shaping_guardrail_count"] = len(policy.Guardrails)
	resolved["response_shaping_instruction_count"] = len(policy.ChannelInstructions)
	return resolved
}

func channelResponseProfileForChannel(channelID string) channelResponseProfile {
	normalizedChannel := normalizeTurnChannelID(channelID)
	if profile, ok := channelResponseProfiles[normalizedChannel]; ok {
		return profile
	}
	return channelResponseProfiles["app"]
}
