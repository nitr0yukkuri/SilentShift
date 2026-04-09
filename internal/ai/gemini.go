package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"silentshift/internal/config"
	"silentshift/internal/logcache"
)

type Client struct {
	httpClient *http.Client
	apiKey     string
	model      string
}

type Analysis struct {
	AwkwardnessScore int      `json:"awkwardnessScore"`
	Theme            string   `json:"theme"`
	Params           []string `json:"params"`
	TauntMessage     string   `json:"tauntMessage"`
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 8 * time.Second},
		apiKey:     cfg.GeminiAPIKey,
		model:      cfg.GeminiModel,
	}
}

func (c *Client) Analyze(ctx context.Context, logs []logcache.Entry) (Analysis, error) {
	if c.apiKey == "" {
		return fallbackAnalysis(logs), nil
	}

	models := []string{canonicalModelName(c.model), "gemini-flash-latest", "gemini-2.5-flash", "gemini-2.0-flash", "gemini-2.5-flash-lite"}
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		model = canonicalModelName(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}

		analysis, err := c.analyzeWithModel(ctx, logs, model)
		if err == nil {
			return analysis, nil
		}
		if !isMissingModelError(err) {
			return Analysis{}, err
		}
		log.Printf("gemini model unavailable, retrying with fallback: %s", model)
	}

	return fallbackAnalysis(logs), nil
}

func (c *Client) analyzeWithModel(ctx context.Context, logs []logcache.Entry, model string) (Analysis, error) {

	type promptPayload struct {
		Logs []logcache.Entry `json:"logs"`
	}

	payload := promptPayload{Logs: logs}
	payloadJSON, _ := json.Marshal(payload)

	instruction := "" +
		"あなたはDiscordの沈黙介入AIです。会話ログから気まずさを解析し、" +
		"JSONのみを返してください。\n" +
		"必須JSON schema: " +
		"{awkwardnessScore:number(0-100),theme:string,params:string[],tauntMessage:string}\n" +
		"tauntMessageは短く煽り気味、ただし攻撃的すぎない。"

	body := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": instruction},
					{"text": string(payloadJSON)},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature":      0.7,
			"responseMimeType": "application/json",
		},
	}

	rawBody, _ := json.Marshal(body)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return Analysis{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(err.Error()), "client.timeout") {
			log.Printf("gemini request timed out, using fallback analysis")
			return fallbackAnalysis(logs), nil
		}
		return Analysis{}, err
	}
	defer resp.Body.Close()

	rspBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusTooManyRequests {
		log.Printf("gemini quota exceeded, using fallback analysis")
		return fallbackAnalysis(logs), nil
	}
	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout || resp.StatusCode >= 500 {
		log.Printf("gemini temporarily unavailable (%d), using fallback analysis", resp.StatusCode)
		return fallbackAnalysis(logs), nil
	}
	if resp.StatusCode >= 400 {
		return Analysis{}, fmt.Errorf("gemini error: %s", strings.TrimSpace(string(rspBody)))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(rspBody, &parsed); err != nil {
		return Analysis{}, err
	}

	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return fallbackAnalysis(logs), nil
	}

	text := strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text)
	var out Analysis
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return fallbackAnalysis(logs), nil
	}

	if out.Theme == "" {
		return fallbackAnalysis(logs), nil
	}

	return out, nil
}

func isMissingModelError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status: \"not_found\"") ||
		strings.Contains(msg, "code\": 404") ||
		strings.Contains(msg, "not found")
}

func canonicalModelName(model string) string {
	m := strings.TrimSpace(strings.ToLower(model))
	switch m {
	case "", "gemini-3.1-flash", "models/gemini-3.1-flash":
		return "gemini-flash-latest"
	default:
		return strings.TrimSpace(model)
	}
}

func fallbackAnalysis(logs []logcache.Entry) Analysis {
	theme := "深夜テンション重力逆転ドッジ"
	if len(logs) > 0 {
		theme = "会話の余韻を殴る段差マラソン"
	}

	return Analysis{
		AwkwardnessScore: 72,
		Theme:            theme,
		Params:           []string{"low_gravity", "banana_friction", "panic_timer_40s"},
		TauntMessage:     "沈黙、検知。言葉は不要。まずは協力してこの地獄を越えろ。",
	}
}
