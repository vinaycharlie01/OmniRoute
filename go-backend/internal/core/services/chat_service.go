package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/omniroute/go-backend/internal/core/domain"
	"github.com/omniroute/go-backend/internal/core/ports/primary"
	"github.com/omniroute/go-backend/internal/core/ports/secondary"
	"go.uber.org/zap"
)

type ChatService struct {
	providerSvc    *ProviderService
	comboSvc       *ComboService
	routingSvc     *RoutingService
	circuitBreaker *CircuitBreakerService
	clientFactory  secondary.LLMClientFactory
	usageRepo      secondary.UsageRepository
	settings       *domain.Settings
	log            *zap.Logger
}

func NewChatService(
	providerSvc *ProviderService,
	comboSvc *ComboService,
	routingSvc *RoutingService,
	cb *CircuitBreakerService,
	clientFactory secondary.LLMClientFactory,
	usageRepo secondary.UsageRepository,
	settings *domain.Settings,
	log *zap.Logger,
) *ChatService {
	return &ChatService{
		providerSvc:    providerSvc,
		comboSvc:       comboSvc,
		routingSvc:     routingSvc,
		circuitBreaker: cb,
		clientFactory:  clientFactory,
		usageRepo:      usageRepo,
		settings:       settings,
		log:            log,
	}
}

// Complete executes a non-streaming chat completion with auto-fallback.
func (s *ChatService) Complete(ctx context.Context, req *domain.ChatRequest, opts *primary.ChatOptions) (*primary.ChatResult, error) {
	requestID := opts.RequestID
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Check if this is a combo model.
	combo, _ := s.comboSvc.GetComboByModel(ctx, req.Model)
	if combo != nil {
		return s.executeCombo(ctx, req, combo, opts, requestID)
	}
	return s.executeSingle(ctx, req, req.Model, opts, requestID)
}

// StreamComplete executes a streaming chat completion.
func (s *ChatService) StreamComplete(ctx context.Context, req *domain.ChatRequest, opts *primary.ChatOptions) (<-chan primary.StreamEvent, error) {
	requestID := opts.RequestID
	if requestID == "" {
		requestID = uuid.New().String()
	}

	ch := make(chan primary.StreamEvent, 64)

	go func() {
		defer close(ch)

		combo, _ := s.comboSvc.GetComboByModel(ctx, req.Model)
		if combo != nil {
			s.streamCombo(ctx, req, combo, opts, requestID, ch)
			return
		}
		s.streamSingle(ctx, req, req.Model, opts, requestID, ch)
	}()

	return ch, nil
}

// Embed creates embeddings.
func (s *ChatService) Embed(ctx context.Context, req *domain.EmbeddingRequest, opts *primary.ChatOptions) (*domain.EmbeddingResponse, error) {
	modelInfo, err := resolveModelProvider(req.Model)
	if err != nil {
		return nil, err
	}

	creds, err := s.providerSvc.GetAvailableCredentials(ctx, modelInfo.Provider, req.Model)
	if err != nil {
		return nil, err
	}
	if len(creds) == 0 {
		return nil, domain.ErrProviderUnavailable
	}

	client, err := s.clientFactory.Create(modelInfo.Provider)
	if err != nil {
		return nil, err
	}

	return client.Embed(ctx, &secondary.LLMEmbedRequest{
		Credentials: creds[0],
		Body:        req,
		TargetModel: req.Model,
		RequestID:   opts.RequestID,
	})
}

func (s *ChatService) executeSingle(
	ctx context.Context,
	req *domain.ChatRequest,
	model string,
	opts *primary.ChatOptions,
	requestID string,
) (*primary.ChatResult, error) {
	modelInfo, err := resolveModelProvider(model)
	if err != nil {
		return nil, err
	}

	if !s.circuitBreaker.CanExecute(ctx, modelInfo.Provider) {
		return nil, domain.NewCircuitOpenError(string(modelInfo.Provider))
	}

	creds, err := s.providerSvc.GetAvailableCredentials(ctx, modelInfo.Provider, model)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, cred := range creds {
		result, err := s.callProvider(ctx, req, cred, model, requestID)
		if err == nil {
			s.circuitBreaker.RecordSuccess(ctx, modelInfo.Provider)
			_ = s.providerSvc.ClearConnectionCooldown(ctx, cred.ConnectionID)
			s.recordUsage(ctx, result.CallLog, opts)
			return result, nil
		}
		lastErr = err
		s.handleProviderError(ctx, err, cred, modelInfo.Provider)
	}

	return nil, fmt.Errorf("all connections failed: %w", lastErr)
}

func (s *ChatService) executeCombo(
	ctx context.Context,
	req *domain.ChatRequest,
	combo *domain.Combo,
	opts *primary.ChatOptions,
	requestID string,
) (*primary.ChatResult, error) {
	if combo.Strategy == domain.StrategyFusion {
		return s.executeFusion(ctx, req, combo, opts, requestID)
	}

	targets, err := s.routingSvc.ResolveTargets(ctx, combo, req)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, target := range targets {
		if !s.circuitBreaker.CanExecute(ctx, target.Provider) {
			continue
		}
		creds, err := s.providerSvc.GetAvailableCredentials(ctx, target.Provider, target.Model)
		if err != nil || len(creds) == 0 {
			continue
		}
		result, err := s.callProvider(ctx, req, creds[0], target.Model, requestID)
		if err == nil {
			s.circuitBreaker.RecordSuccess(ctx, target.Provider)
			s.recordUsage(ctx, result.CallLog, opts)
			return result, nil
		}
		lastErr = err
		s.handleProviderError(ctx, err, creds[0], target.Provider)
	}

	return nil, fmt.Errorf("all combo targets failed: %w", lastErr)
}

func (s *ChatService) executeFusion(
	ctx context.Context,
	req *domain.ChatRequest,
	combo *domain.Combo,
	opts *primary.ChatOptions,
	requestID string,
) (*primary.ChatResult, error) {
	targets := combo.ActiveNodes()
	if len(targets) == 0 {
		return nil, domain.ErrAllProvidersFailed
	}

	type candidate struct {
		content string
		model   string
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	candidates := make([]candidate, 0, len(targets))

	for _, target := range targets {
		target := target
		wg.Add(1)
		go func() {
			defer wg.Done()
			creds, err := s.providerSvc.GetAvailableCredentials(ctx, target.Provider, target.Model)
			if err != nil || len(creds) == 0 {
				return
			}
			result, err := s.callProvider(ctx, req, creds[0], target.Model, requestID)
			if err != nil || result.Response == nil || len(result.Response.Choices) == 0 {
				return
			}
			content := extractContent(result.Response.Choices[0].Message)
			mu.Lock()
			candidates = append(candidates, candidate{content: content, model: target.Model})
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(candidates) == 0 {
		return nil, domain.ErrAllProvidersFailed
	}
	if len(candidates) == 1 {
		return s.callProvider(ctx, req, nil, candidates[0].model, requestID)
	}

	// Build fusion prompt for the judge.
	judgeModel := combo.FusionJudgeModel
	if judgeModel == "" && len(targets) > 0 {
		judgeModel = targets[0].Model
	}

	var fusionPrompt strings.Builder
	fusionPrompt.WriteString("You are a synthesis judge. Multiple AI models answered the following question. Synthesize the best final answer:\n\n")
	fusionPrompt.WriteString("Original question: ")
	for _, msg := range req.Messages {
		if msg.Role == domain.RoleUser {
			fusionPrompt.WriteString(extractContent(msg))
			break
		}
	}
	fusionPrompt.WriteString("\n\nCandidate answers:\n")
	for i, c := range candidates {
		fusionPrompt.WriteString(fmt.Sprintf("\n--- Answer %d (from %s) ---\n%s\n", i+1, c.model, c.content))
	}
	fusionPrompt.WriteString("\nProvide a single synthesized answer.")

	judgeReq := *req
	judgeReq.Model = judgeModel
	judgeReq.Messages = []domain.Message{
		{Role: domain.RoleUser, Content: fusionPrompt.String()},
	}
	return s.executeSingle(ctx, &judgeReq, judgeModel, opts, requestID)
}

func (s *ChatService) streamSingle(
	ctx context.Context,
	req *domain.ChatRequest,
	model string,
	opts *primary.ChatOptions,
	requestID string,
	ch chan<- primary.StreamEvent,
) {
	modelInfo, err := resolveModelProvider(model)
	if err != nil {
		ch <- primary.StreamEvent{Error: err}
		return
	}

	if !s.circuitBreaker.CanExecute(ctx, modelInfo.Provider) {
		ch <- primary.StreamEvent{Error: domain.NewCircuitOpenError(string(modelInfo.Provider))}
		return
	}

	creds, err := s.providerSvc.GetAvailableCredentials(ctx, modelInfo.Provider, model)
	if err != nil {
		ch <- primary.StreamEvent{Error: err}
		return
	}

	client, err := s.clientFactory.Create(modelInfo.Provider)
	if err != nil {
		ch <- primary.StreamEvent{Error: err}
		return
	}

	reader, err := client.ChatStream(ctx, &secondary.LLMRequest{
		Credentials: creds[0],
		Body:        req,
		TargetModel: model,
		RequestID:   requestID,
	})
	if err != nil {
		ch <- primary.StreamEvent{Error: err}
		return
	}
	defer reader.Close()

	s.pipeSSE(ctx, reader, ch)
}

func (s *ChatService) streamCombo(
	ctx context.Context,
	req *domain.ChatRequest,
	combo *domain.Combo,
	opts *primary.ChatOptions,
	requestID string,
	ch chan<- primary.StreamEvent,
) {
	targets, err := s.routingSvc.ResolveTargets(ctx, combo, req)
	if err != nil {
		ch <- primary.StreamEvent{Error: err}
		return
	}

	for _, target := range targets {
		if !s.circuitBreaker.CanExecute(ctx, target.Provider) {
			continue
		}
		creds, err := s.providerSvc.GetAvailableCredentials(ctx, target.Provider, target.Model)
		if err != nil || len(creds) == 0 {
			continue
		}
		client, err := s.clientFactory.Create(target.Provider)
		if err != nil {
			continue
		}
		reader, err := client.ChatStream(ctx, &secondary.LLMRequest{
			Credentials: creds[0],
			Body:        req,
			TargetModel: target.Model,
			RequestID:   requestID,
		})
		if err != nil {
			s.handleProviderError(ctx, err, creds[0], target.Provider)
			continue
		}
		defer reader.Close()
		s.pipeSSE(ctx, reader, ch)
		return
	}
	ch <- primary.StreamEvent{Error: domain.ErrAllProvidersFailed}
}

func (s *ChatService) pipeSSE(ctx context.Context, reader io.Reader, ch chan<- primary.StreamEvent) {
	scanner := bufio.NewScanner(reader)
	heartbeat := time.NewTicker(time.Duration(s.settings.SSEHeartbeatIntervalMs) * time.Millisecond)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			ch <- primary.StreamEvent{Error: ctx.Err()}
			return
		case <-heartbeat.C:
			ch <- primary.StreamEvent{Data: []byte(": heartbeat\n\n")}
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					ch <- primary.StreamEvent{Error: err}
				}
				ch <- primary.StreamEvent{Done: true}
				return
			}
			line := scanner.Bytes()
			if len(line) == 0 {
				ch <- primary.StreamEvent{Data: []byte("\n")}
				continue
			}
			out := make([]byte, len(line)+1)
			copy(out, line)
			out[len(line)] = '\n'
			ch <- primary.StreamEvent{Data: out}
		}
	}
}

func (s *ChatService) callProvider(
	ctx context.Context,
	req *domain.ChatRequest,
	cred *domain.ProviderCredentials,
	model string,
	requestID string,
) (*primary.ChatResult, error) {
	if cred == nil {
		return nil, domain.ErrProviderUnavailable
	}

	client, err := s.clientFactory.Create(cred.Provider)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := client.ChatComplete(ctx, &secondary.LLMRequest{
		Credentials: cred,
		Body:        req,
		TargetModel: model,
		RequestID:   requestID,
	})
	if err != nil {
		return nil, err
	}

	callLog := &domain.CallLog{
		ID:           uuid.New().String(),
		RequestID:    requestID,
		Provider:     cred.Provider,
		Model:        model,
		ConnectionID: cred.ConnectionID,
		StatusCode:   resp.StatusCode,
		DurationMs:   time.Since(start).Milliseconds(),
		Timestamp:    time.Now(),
	}
	if resp.Response != nil {
		callLog.InputTokens = resp.Response.Usage.PromptTokens
		callLog.OutputTokens = resp.Response.Usage.CompletionTokens
		callLog.TotalTokens = resp.Response.Usage.TotalTokens
	}

	return &primary.ChatResult{
		Response:     resp.Response,
		CallLog:      callLog,
		Provider:     cred.Provider,
		Model:        model,
		ConnectionID: cred.ConnectionID,
	}, nil
}

func (s *ChatService) handleProviderError(
	ctx context.Context,
	err error,
	cred *domain.ProviderCredentials,
	provider domain.ProviderType,
) {
	domErr, ok := err.(*domain.DomainError)
	if !ok {
		return
	}
	switch domErr.Code {
	case "rate_limit_exceeded":
		_ = s.providerSvc.MarkConnectionCooled(ctx, cred.ConnectionID,
			s.settings.ResilienceSettings.APIKeyBaseCooldownMs, 0, "rate_limit", "429", domErr.Message)
	case "provider_error":
		s.circuitBreaker.RecordFailure(ctx, provider, 500)
	}
}

func (s *ChatService) recordUsage(ctx context.Context, log *domain.CallLog, opts *primary.ChatOptions) {
	if log == nil {
		return
	}
	log.APIKeyID = opts.APIKeyID
	_ = s.usageRepo.SaveCallLog(ctx, log)
}

func resolveModelProvider(model string) (*domain.ModelInfo, error) {
	info := &domain.ModelInfo{Model: model}

	switch {
	case strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3"):
		info.Provider = domain.ProviderTypeOpenAI
		info.Format = domain.FormatOpenAI
	case strings.HasPrefix(model, "claude"):
		info.Provider = domain.ProviderTypeAnthropic
		info.Format = domain.FormatAnthropic
	case strings.HasPrefix(model, "gemini"):
		info.Provider = domain.ProviderTypeGemini
		info.Format = domain.FormatGemini
	default:
		info.Provider = domain.ProviderTypeOpenAI
		info.Format = domain.FormatOpenAI
	}
	return info, nil
}

func extractContent(msg any) string {
	switch v := msg.(type) {
	case domain.Message:
		switch c := v.Content.(type) {
		case string:
			return c
		case []any:
			var b strings.Builder
			for _, part := range c {
				if m, ok := part.(map[string]any); ok {
					if t, ok := m["text"].(string); ok {
						b.WriteString(t)
					}
				}
			}
			return b.String()
		}
	}
	return ""
}

func marshalJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

var _ = bytes.NewBuffer
var _ = marshalJSON
