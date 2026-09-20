package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var ErrNotFound = errors.New("config: not found")

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) do(ctx context.Context, method string, parts []string, body any) (*http.Response, error) {
	u, err := url.JoinPath(c.baseURL, parts...)
	if err != nil {
		return nil, fmt.Errorf("build url: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	return resp, nil
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
	return nil
}

func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
}

func (c *Client) Get(ctx context.Context, namespace, key string) (string, error) {
	resp, err := c.do(ctx, http.MethodGet, []string{"api", "v1", "config", namespace, key}, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return "", err
	}

	var out struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return out.Value, nil
}

func (c *Client) Set(ctx context.Context, namespace, key, value string) error {
	body := map[string]string{"value": value}
	resp, err := c.do(ctx, http.MethodPost, []string{"api", "v1", "config", namespace, key}, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return err
	}
	drain(resp)
	return nil
}

func (c *Client) List(ctx context.Context, namespace string) (map[string]string, error) {
	resp, err := c.do(ctx, http.MethodGet, []string{"api", "v1", "config", namespace}, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	var out struct {
		Configs map[string]string `json:"configs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if out.Configs == nil {
		return map[string]string{}, nil
	}
	return out.Configs, nil
}

func (c *Client) Delete(ctx context.Context, namespace, key string) error {
	resp, err := c.do(ctx, http.MethodDelete, []string{"api", "v1", "config", namespace, key}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return err
	}
	drain(resp)
	return nil
}
