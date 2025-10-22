package agentbill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// Config represents the AgentBill SDK configuration
type Config struct {
	APIKey     string
	BaseURL    string
	CustomerID string
	Debug      bool
}

// Client is the main AgentBill SDK client
type Client struct {
	config Config
	tracer *Tracer
}

// Init initializes a new AgentBill client
func Init(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://uenhjwdtnxtchlmqarjo.supabase.co"
	}
	return &Client{
		config: config,
		tracer: NewTracer(config),
	}
}

// OpenAIWrapper wraps OpenAI client calls
type OpenAIWrapper struct {
	client *Client
}

// WrapOpenAI wraps an OpenAI client for tracking
func (c *Client) WrapOpenAI() *OpenAIWrapper {
	return &OpenAIWrapper{client: c}
}

// ChatCompletion tracks an OpenAI chat completion call
func (w *OpenAIWrapper) ChatCompletion(ctx context.Context, model string, messages []map[string]string) (map[string]interface{}, error) {
	startTime := time.Now()

	span := w.client.tracer.StartSpan("openai.chat.completion", map[string]interface{}{
		"model":    model,
		"provider": "openai",
	})

	defer func() {
		latency := time.Since(startTime).Milliseconds()
		span.SetAttribute("latency_ms", latency)
		span.End()
	}()

	// Build request payload
	requestBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// Make actual OpenAI API call
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		err := fmt.Errorf("OPENAI_API_KEY environment variable not set")
		span.SetStatus(1, err.Error())
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("OpenAI API returned status: %d", resp.StatusCode)
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// Extract token usage
	if usage, ok := response["usage"].(map[string]interface{}); ok {
		if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
			span.SetAttribute("response.prompt_tokens", int(promptTokens))
		}
		if completionTokens, ok := usage["completion_tokens"].(float64); ok {
			span.SetAttribute("response.completion_tokens", int(completionTokens))
		}
		if totalTokens, ok := usage["total_tokens"].(float64); ok {
			span.SetAttribute("response.total_tokens", int(totalTokens))
		}
	}

	span.SetStatus(0, "")
	return response, nil
}

// Signal represents a custom event with revenue
type Signal struct {
	EventName  string                 `json:"event_name"`
	Revenue    float64                `json:"revenue"`
	CustomerID string                 `json:"customer_id"`
	Timestamp  int64                  `json:"timestamp"`
	Data       map[string]interface{} `json:"data"`
}

// TrackSignal tracks a custom signal/event with revenue
func (c *Client) TrackSignal(ctx context.Context, signal Signal) error {
	url := fmt.Sprintf("%s/functions/v1/record-signals", c.config.BaseURL)
	
	signal.CustomerID = c.config.CustomerID
	signal.Timestamp = time.Now().Unix()
	if signal.Data == nil {
		signal.Data = make(map[string]interface{})
	}
	
	jsonData, err := json.Marshal(signal)
	if err != nil {
		return err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if c.config.Debug {
		fmt.Printf("[AgentBill] Signal tracked: %s, revenue: $%.2f\n", signal.EventName, signal.Revenue)
	}
	
	return nil
}

// Flush flushes pending telemetry data
func (c *Client) Flush(ctx context.Context) error {
	return c.tracer.Flush(ctx)
}

// Tracer handles OpenTelemetry tracing
type Tracer struct {
	config Config
	spans  []*Span
}

// Span represents an OpenTelemetry span
type Span struct {
	Name       string
	TraceID    string
	SpanID     string
	Attributes map[string]interface{}
	StartTime  int64
	EndTime    int64
	Status     map[string]interface{}
}

// NewTracer creates a new tracer
func NewTracer(config Config) *Tracer {
	return &Tracer{
		config: config,
		spans:  make([]*Span, 0),
	}
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(name string, attributes map[string]interface{}) *Span {
	traceID := uuid.New().String()
	spanID := uuid.New().String()[:16]

	attributes["service.name"] = "agentbill-go-sdk"
	if t.config.CustomerID != "" {
		attributes["customer.id"] = t.config.CustomerID
	}

	span := &Span{
		Name:       name,
		TraceID:    traceID,
		SpanID:     spanID,
		Attributes: attributes,
		StartTime:  time.Now().UnixNano(),
		Status:     map[string]interface{}{"code": 0},
	}

	t.spans = append(t.spans, span)
	return span
}

// SetAttribute sets an attribute on the span
func (s *Span) SetAttribute(key string, value interface{}) {
	s.Attributes[key] = value
}

// SetStatus sets the status of the span
func (s *Span) SetStatus(code int, message string) {
	s.Status = map[string]interface{}{
		"code":    code,
		"message": message,
	}
}

// End ends the span
func (s *Span) End() {
	s.EndTime = time.Now().UnixNano()
}

// Flush sends spans to AgentBill
func (t *Tracer) Flush(ctx context.Context) error {
	if len(t.spans) == 0 {
		return nil
	}

	payload := t.buildOTLPPayload()
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/functions/v1/otel-collector", t.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.config.APIKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if t.config.Debug {
		fmt.Printf("AgentBill flush: %d\n", resp.StatusCode)
	}

	if resp.StatusCode == 200 {
		t.spans = make([]*Span, 0)
	}

	return nil
}

func (t *Tracer) buildOTLPPayload() map[string]interface{} {
	spans := make([]map[string]interface{}, len(t.spans))
	for i, span := range t.spans {
		spans[i] = t.spanToOTLP(span)
	}

	return map[string]interface{}{
		"resourceSpans": []map[string]interface{}{
			{
				"resource": map[string]interface{}{
					"attributes": []map[string]interface{}{
						{"key": "service.name", "value": map[string]interface{}{"stringValue": "agentbill-go-sdk"}},
						{"key": "service.version", "value": map[string]interface{}{"stringValue": "1.0.0"}},
					},
				},
				"scopeSpans": []map[string]interface{}{
					{
						"scope": map[string]interface{}{"name": "agentbill", "version": "1.0.0"},
						"spans": spans,
					},
				},
			},
		},
	}
}

func (t *Tracer) spanToOTLP(span *Span) map[string]interface{} {
	attributes := make([]map[string]interface{}, 0, len(span.Attributes))
	for k, v := range span.Attributes {
		attributes = append(attributes, map[string]interface{}{
			"key":   k,
			"value": t.valueToOTLP(v),
		})
	}

	endTime := span.EndTime
	if endTime == 0 {
		endTime = time.Now().UnixNano()
	}

	return map[string]interface{}{
		"traceId":           span.TraceID,
		"spanId":            span.SpanID,
		"name":              span.Name,
		"kind":              1,
		"startTimeUnixNano": fmt.Sprintf("%d", span.StartTime),
		"endTimeUnixNano":   fmt.Sprintf("%d", endTime),
		"attributes":        attributes,
		"status":            span.Status,
	}
}

func (t *Tracer) valueToOTLP(value interface{}) map[string]interface{} {
	switch v := value.(type) {
	case string:
		return map[string]interface{}{"stringValue": v}
	case int, int64:
		return map[string]interface{}{"intValue": v}
	case bool:
		return map[string]interface{}{"boolValue": v}
	default:
		return map[string]interface{}{"stringValue": fmt.Sprintf("%v", v)}
	}
}
