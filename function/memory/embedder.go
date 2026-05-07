package memory

import (
	"300Bot/conf"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Embedder interface {
	Embed(text string) ([]float32, error)
}

type OpenAICompatibleEmbedder struct {
	client *http.Client
	apiURL string
	apiKey string
	model  string
}

type DashScopeMultimodalEmbedder struct {
	client    *http.Client
	apiURL    string
	apiKey    string
	model     string
	dimension int
}

func NewOpenAICompatibleEmbedder(cfg conf.MemoryConfig, timeout time.Duration) *OpenAICompatibleEmbedder {
	return &OpenAICompatibleEmbedder{
		client: &http.Client{Timeout: timeout},
		apiURL: strings.TrimSpace(cfg.EmbeddingApiUrl),
		apiKey: strings.TrimSpace(cfg.EmbeddingApiKey),
		model:  strings.TrimSpace(cfg.EmbeddingModel),
	}
}

func NewDashScopeMultimodalEmbedder(cfg conf.MemoryConfig, timeout time.Duration) *DashScopeMultimodalEmbedder {
	return &DashScopeMultimodalEmbedder{
		client:    &http.Client{Timeout: timeout},
		apiURL:    strings.TrimSpace(cfg.EmbeddingApiUrl),
		apiKey:    strings.TrimSpace(cfg.EmbeddingApiKey),
		model:     strings.TrimSpace(cfg.EmbeddingModel),
		dimension: cfg.EmbeddingDimension,
	}
}

func NewEmbedder(cfg conf.MemoryConfig, timeout time.Duration) Embedder {
	provider := strings.ToLower(strings.TrimSpace(cfg.EmbeddingProvider))
	if provider == "ali" || provider == "dashscope" {
		return NewDashScopeMultimodalEmbedder(cfg, timeout)
	}
	return NewOpenAICompatibleEmbedder(cfg, timeout)
}

func (e *OpenAICompatibleEmbedder) Embed(text string) ([]float32, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("embedding text 不能为空")
	}
	if e.apiURL == "" {
		return nil, fmt.Errorf("embeddingApiUrl 未配置")
	}
	if e.model == "" {
		return nil, fmt.Errorf("embeddingModel 未配置")
	}

	payload := map[string]interface{}{
		"model": e.model,
		"input": text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, e.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding 请求失败 status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err = json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("embedding 响应为空")
	}

	result := make([]float32, len(parsed.Data[0].Embedding))
	for i, v := range parsed.Data[0].Embedding {
		result[i] = float32(v)
	}
	return result, nil
}

func (e *DashScopeMultimodalEmbedder) Embed(text string) ([]float32, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("embedding text 不能为空")
	}
	if e.apiURL == "" {
		return nil, fmt.Errorf("embeddingApiUrl 未配置")
	}
	if e.model == "" {
		return nil, fmt.Errorf("embeddingModel 未配置")
	}
	if e.dimension <= 0 {
		return nil, fmt.Errorf("embeddingDimension 必须 > 0")
	}

	payload := map[string]interface{}{
		"model": e.model,
		"input": map[string]interface{}{
			"contents": []map[string]interface{}{
				{"text": text},
			},
		},
		"parameters": map[string]interface{}{
			"dimension": e.dimension,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, e.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding 请求失败 status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Output struct {
			Embeddings []struct {
				Embedding []float64 `json:"embedding"`
				Type      string    `json:"type"`
			} `json:"embeddings"`
		} `json:"output"`
	}
	if err = json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}

	var embedding []float64
	for _, item := range parsed.Output.Embeddings {
		if strings.EqualFold(strings.TrimSpace(item.Type), "text") && len(item.Embedding) > 0 {
			embedding = item.Embedding
			break
		}
		if embedding == nil && len(item.Embedding) > 0 {
			embedding = item.Embedding
		}
	}
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding 响应为空")
	}

	result := make([]float32, len(embedding))
	for i, v := range embedding {
		result[i] = float32(v)
	}
	return result, nil
}
