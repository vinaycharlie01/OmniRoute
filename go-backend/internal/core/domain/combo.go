package domain

import "time"

type ComboStrategy string

const (
	StrategyPriority        ComboStrategy = "priority"
	StrategyWeighted        ComboStrategy = "weighted"
	StrategyRoundRobin      ComboStrategy = "round-robin"
	StrategyFillFirst       ComboStrategy = "fill-first"
	StrategyCostOptimized   ComboStrategy = "cost-optimized"
	StrategyRandom          ComboStrategy = "random"
	StrategyLeastUsed       ComboStrategy = "least-used"
	StrategyP2C             ComboStrategy = "P2C"
	StrategyResetAware      ComboStrategy = "reset-aware"
	StrategyResetWindow     ComboStrategy = "reset-window"
	StrategyHeadroom        ComboStrategy = "headroom"
	StrategyStrictRandom    ComboStrategy = "strict-random"
	StrategyAuto            ComboStrategy = "auto"
	StrategyLKGP            ComboStrategy = "lkgp"
	StrategyContextOptimized ComboStrategy = "context-optimized"
	StrategyContextRelay    ComboStrategy = "context-relay"
	StrategyFusion          ComboStrategy = "fusion"
)

type ComboNode struct {
	ID           string       `json:"id"`
	ComboID      string       `json:"comboId"`
	ConnectionID string       `json:"connectionId"`
	Provider     ProviderType `json:"provider"`
	Model        string       `json:"model"`
	Weight       int          `json:"weight"`
	Priority     int          `json:"priority"`
	IsActive     bool         `json:"isActive"`
	MaxTokens    int          `json:"maxTokens,omitempty"`
	Temperature  *float64     `json:"temperature,omitempty"`
}

type Combo struct {
	ID                    string        `json:"id"`
	Name                  string        `json:"name"`
	Model                 string        `json:"model"`
	Strategy              ComboStrategy `json:"strategy"`
	IsActive              bool          `json:"isActive"`
	Nodes                 []ComboNode   `json:"nodes"`
	MaxRetries            int           `json:"maxRetries"`
	RetryDelayMs          int           `json:"retryDelayMs"`
	FallbackDelayMs       int           `json:"fallbackDelayMs"`
	TimeoutMs             int           `json:"timeoutMs"`
	HealthCheckEnabled    bool          `json:"healthCheckEnabled"`
	MaxComboDepth         int           `json:"maxComboDepth"`
	TrackMetrics          bool          `json:"trackMetrics"`
	ReasoningTokenBuffer  bool          `json:"reasoningTokenBufferEnabled"`
	FusionJudgeModel      string        `json:"fusionJudgeModel,omitempty"`
	FusionMaxCandidates   int           `json:"fusionMaxCandidates,omitempty"`
	CreatedAt             time.Time     `json:"createdAt"`
	UpdatedAt             time.Time     `json:"updatedAt"`
}

func (c *Combo) ActiveNodes() []ComboNode {
	active := make([]ComboNode, 0, len(c.Nodes))
	for _, n := range c.Nodes {
		if n.IsActive {
			active = append(active, n)
		}
	}
	return active
}

type ComboTarget struct {
	ConnectionID string
	Provider     ProviderType
	Model        string
	Weight       int
	Priority     int
}

type AutoComboScore struct {
	ConnectionID    string
	Provider        ProviderType
	Model           string
	Score           float64
	LatencyMs       float64
	SuccessRate     float64
	CostPerToken    float64
	ContextCapacity float64
	Headroom        float64
}
