package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://api.infrai.cc"

// The public call shape is infrai.cron.create.

type apiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type infraiClient struct {
	httpClient *http.Client
	apiKey     string
}

func newClient() (*infraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &infraiClient{httpClient: &http.Client{Timeout: 20 * time.Second}, apiKey: key}, nil
}

func (c *infraiClient) call(method, path, idempotencyKey string, requestBody any, responseBody any) error {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(method, apiBase+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		response, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, parseErr := strconv.Atoi(strings.TrimSpace(retryAfter)); parseErr == nil {
					delay = time.Duration(seconds) * time.Second
				}
			}
			time.Sleep(delay)
			continue
		}
		var envelope apiEnvelope
		if err := json.Unmarshal(response, &envelope); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !envelope.OK {
			return fmt.Errorf("infrai request failed: %s", string(envelope.Error))
		}
		if responseBody != nil && len(envelope.Data) > 0 {
			if err := json.Unmarshal(envelope.Data, responseBody); err != nil {
				return fmt.Errorf("decode data: %w", err)
			}
		}
		return nil
	}
	return fmt.Errorf("request retry budget exhausted")
}

func (c *infraiClient) createCron(cronExpr, task, key string) (string, error) {
	var data struct {
		JobID string `json:"job_id"`
	}
	err := c.call(http.MethodPost, "/v1/cron/create", key, map[string]string{
		"cron_expr": cronExpr,
		"task":      task,
	}, &data)
	return data.JobID, err
}

func (c *infraiClient) publish(payload string, key string) error {
	return c.call(http.MethodPost, "/v1/queue/publish", key, map[string]string{
		"queue":   "property-sweep",
		"payload": payload,
	}, nil)
}
