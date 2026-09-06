// Package providerconfig is the LLM Provider Config abstraction layer.
//
// A vendor provider (e.g. MiniMax, GLM) stores ONE API key plus a set of
// per-protocol endpoint variants. This package maps an agent adapter
// (claude / codex / opencode) onto the variant that vendor serves, so a
// single API key powers every agent and newly added models (provider.models)
// work everywhere without further code changes.
package providerconfig

import (
	"fmt"
	"sort"

	"github.com/variableway/innate-aiswitcher/internal/store"
)

// Wire protocols a variant can speak. They mirror providers.api_protocol.
const (
	ProtocolAnthropic       = "anthropic"
	ProtocolOpenAIChat      = "openai_chat"
	ProtocolOpenAIResponses = "openai_responses"
)

// agentProtocols lists, per agent adapter, the wire protocols it can use in
// preference order. Codex accepts the Responses API first and falls back to
// chat-completions; opencode (and other openai_env agents) speak chat.
var agentProtocols = map[string][]string{
	"claude":     {ProtocolAnthropic},
	"codex":      {ProtocolOpenAIResponses, ProtocolOpenAIChat},
	"openai_env": {ProtocolOpenAIChat},
}

// AgentProtocols returns the preferred wire protocols for an agent adapter.
func AgentProtocols(adapter string) ([]string, bool) {
	protocols, ok := agentProtocols[adapter]
	return protocols, ok
}

// SupportedProtocols enumerates all protocols some agent can speak.
func SupportedProtocols() []string {
	return []string{ProtocolAnthropic, ProtocolOpenAIChat, ProtocolOpenAIResponses}
}

// SupportedAgents returns the agent adapters a protocol can serve.
func SupportedAgents(protocol string) []string {
	var agents []string
	for adapter, protocols := range agentProtocols {
		for _, candidate := range protocols {
			if candidate == protocol {
				agents = append(agents, adapter)
				break
			}
		}
	}
	sort.Strings(agents)
	return agents
}

// SupportsAgent reports whether the provider has an endpoint the agent
// adapter can consume: either a matching variant, or (for legacy
// single-protocol providers) a top-level protocol the adapter accepts.
func SupportsAgent(provider store.Provider, adapter string) bool {
	protocols, ok := agentProtocols[adapter]
	if !ok {
		return false
	}
	if len(provider.Variants) == 0 {
		return contains(protocols, provider.APIProtocol)
	}
	for _, protocol := range protocols {
		if _, ok := provider.Variants[protocol]; ok {
			return true
		}
	}
	return false
}

// Resolve projects a vendor provider onto the endpoint a specific agent
// adapter needs. For variant providers the matching variant overrides
// base_url/api_protocol/endpoints/capabilities; the API key, model list and
// default model stay shared. Legacy single-protocol providers pass through
// unchanged as long as the adapter can speak their protocol. Adapters not in
// the registry (custom builders registered via adapter.Register) pass through
// untouched so extensions keep working.
func Resolve(provider store.Provider, adapter string) (store.Provider, error) {
	protocols, registered := agentProtocols[adapter]
	if len(provider.Variants) == 0 {
		if !registered || contains(protocols, provider.APIProtocol) {
			return provider, nil
		}
		return store.Provider{}, fmt.Errorf(
			"provider %s (%s) does not support agent %s (needs %s)",
			provider.Slug, provider.APIProtocol, adapter, joinProtocols(protocols))
	}
	for _, protocol := range protocols {
		variant, ok := provider.Variants[protocol]
		if !ok {
			continue
		}
		resolved := provider
		resolved.APIProtocol = protocol
		resolved.BaseURL = variant.BaseURL
		if len(variant.Endpoints) > 0 {
			resolved.Endpoints = variant.Endpoints
		}
		if len(variant.Capabilities) > 0 {
			resolved.Capabilities = mergeCapabilities(provider.Capabilities, variant.Capabilities)
		}
		return resolved, nil
	}
	return store.Provider{}, fmt.Errorf(
		"provider %s has no endpoint for agent %s (needs %s, has %s)",
		provider.Slug, adapter, joinProtocols(protocols), joinProtocols(variantProtocols(provider.Variants)))
}

// ResolveDefault projects a vendor provider onto a deterministic endpoint for
// protocol-agnostic consumers such as connectivity tests: the top-level
// api_protocol/base_url when set, otherwise the preferred variant.
func ResolveDefault(provider store.Provider) store.Provider {
	if len(provider.Variants) == 0 || (provider.BaseURL != "" && provider.APIProtocol != "") {
		return provider
	}
	order := append([]string{provider.APIProtocol}, ProtocolOpenAIChat, ProtocolOpenAIResponses, ProtocolAnthropic)
	for _, protocol := range order {
		if protocol == "" {
			continue
		}
		if variant, ok := provider.Variants[protocol]; ok {
			resolved, err := Resolve(provider, adapterForProtocol(protocol))
			if err == nil {
				return resolved
			}
			resolved = provider
			resolved.APIProtocol = protocol
			resolved.BaseURL = variant.BaseURL
			if len(variant.Endpoints) > 0 {
				resolved.Endpoints = variant.Endpoints
			}
			return resolved
		}
	}
	return provider
}

func adapterForProtocol(protocol string) string {
	switch protocol {
	case ProtocolAnthropic:
		return "claude"
	case ProtocolOpenAIResponses:
		return "codex"
	default:
		return "openai_env"
	}
}

func variantProtocols(variants map[string]store.ProviderVariant) []string {
	protocols := make([]string, 0, len(variants))
	for protocol := range variants {
		protocols = append(protocols, protocol)
	}
	sort.Strings(protocols)
	return protocols
}

func mergeCapabilities(base, overlay map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(base)+len(overlay))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		merged[key] = value
	}
	return merged
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func joinProtocols(protocols []string) string {
	result := ""
	for i, protocol := range protocols {
		if i > 0 {
			result += "|"
		}
		result += protocol
	}
	return result
}
