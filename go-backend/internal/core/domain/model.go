package domain

type ModelFormat string

const (
	FormatOpenAI    ModelFormat = "openai"
	FormatAnthropic ModelFormat = "anthropic"
	FormatGemini    ModelFormat = "gemini"
	FormatOllama    ModelFormat = "ollama"
)

type ModelCapability string

const (
	CapabilityChat        ModelCapability = "chat"
	CapabilityEmbeddings  ModelCapability = "embeddings"
	CapabilityVision      ModelCapability = "vision"
	CapabilityTools       ModelCapability = "tools"
	CapabilityReasoning   ModelCapability = "reasoning"
	CapabilityCodeExec    ModelCapability = "code_execution"
	CapabilityImageGen    ModelCapability = "image_generation"
	CapabilityAudioSpeech ModelCapability = "audio_speech"
	CapabilityAudioSTT    ModelCapability = "audio_transcription"
)

type Model struct {
	ID               string            `json:"id"`
	Provider         ProviderType      `json:"provider"`
	Name             string            `json:"name"`
	DisplayName      string            `json:"displayName"`
	Format           ModelFormat       `json:"format"`
	Capabilities     []ModelCapability `json:"capabilities"`
	ContextWindow    int               `json:"contextWindow"`
	MaxOutputTokens  int               `json:"maxOutputTokens"`
	InputCostPerMTok float64           `json:"inputCostPerMTok"`
	OutputCostPerMTok float64          `json:"outputCostPerMTok"`
	IsDeprecated     bool              `json:"isDeprecated"`
	DeprecatedBy     string            `json:"deprecatedBy,omitempty"`
	IsActive         bool              `json:"isActive"`
	Tags             []string          `json:"tags,omitempty"`
}

func (m *Model) HasCapability(c ModelCapability) bool {
	for _, cap := range m.Capabilities {
		if cap == c {
			return true
		}
	}
	return false
}

func (m *Model) SupportsVision() bool      { return m.HasCapability(CapabilityVision) }
func (m *Model) SupportsTools() bool       { return m.HasCapability(CapabilityTools) }
func (m *Model) SupportsReasoning() bool   { return m.HasCapability(CapabilityReasoning) }

func (m *Model) EstimateCost(inputTokens, outputTokens int) float64 {
	return float64(inputTokens)*m.InputCostPerMTok/1_000_000 +
		float64(outputTokens)*m.OutputCostPerMTok/1_000_000
}

type ModelInfo struct {
	Model        string       `json:"model"`
	Provider     ProviderType `json:"provider"`
	Format       ModelFormat  `json:"format"`
	IsCombo      bool         `json:"isCombo"`
	ComboID      string       `json:"comboId,omitempty"`
}
