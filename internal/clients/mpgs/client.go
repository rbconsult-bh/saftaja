package mpgs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	defaultHTTPClientTimeout = 30 * time.Second // MPGS can be slower than typical APIs
)

// Option configures the client
type Option func(*client)

// WithHTTPClientTimeout returns an Option that sets the HTTP client timeout
func WithHTTPClientTimeout(timeout time.Duration) Option {
	return func(c *client) {
		c.hc.Timeout = timeout
	}
}

// WithBasicAuth sets basic authentication credentials
// For MPGS, just pass your merchantID and API password
// The client will automatically format it as "merchant.{merchantId}"
func WithBasicAuth(merchantID, apiPassword string) Option {
	return func(c *client) {
		c.username = fmt.Sprintf("merchant.%s", merchantID)
		c.password = apiPassword
	}
}

type client struct {
	baseURL  string
	username string
	password string
	hc       *http.Client
}

// New creates a new MPGS client
func New(baseURL string, opts ...Option) Client {
	c := &client{
		baseURL: baseURL,
		hc: &http.Client{
			Timeout:   defaultHTTPClientTimeout,
			Transport: otelhttp.NewTransport(cleanhttp.DefaultTransport()),
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// do performs an HTTP request with authentication
func (c *client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s%s", c.baseURL, path), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.hc.Do(req)
}

// decodeResponse decodes the API response using generics
func decodeResponse[T any](resp *http.Response, expectedStatus int) (*Response[T], error) {
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		var errResponse ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResponse); err != nil {
			return nil, fmt.Errorf("failed to decode error response: %w, response status: %d", err, resp.StatusCode)
		}
		return nil, errResponse
	}

	var data T
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Response[T]{Data: data}, nil
}

// CreateSession creates a payment session
func (c *client) CreateSession(ctx context.Context, merchantID string, req CreateSessionRequest) (*Response[CreateSessionResponse], error) {
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode request body: %w", err)
	}

	path := fmt.Sprintf("/version/%s/merchant/%s/session", APIVersion, merchantID)

	resp, err := c.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return decodeResponse[CreateSessionResponse](resp, http.StatusCreated)
}
