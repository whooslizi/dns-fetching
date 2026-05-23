package doh

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type Client struct {
	httpClient *http.Client
	primary    Provider
	fallback   *Provider
	mu         sync.RWMutex
	metrics    Metrics
}

type Metrics struct {
	TotalQueries  uint64
	FailedQueries uint64
	AvgLatencyMs  float64
	latencySum    float64
	latencyCount  uint64
}

func NewClient(primary Provider, fallback *Provider) *Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		ForceAttemptHTTP2:   true,
	}
	return &Client{
		httpClient: &http.Client{Transport: transport, Timeout: 5 * time.Second},
		primary:    primary,
		fallback:   fallback,
	}
}

func (c *Client) Exchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	c.mu.Lock()
	c.metrics.TotalQueries++
	c.mu.Unlock()

	start := time.Now()

	resp, err := c.doExchange(ctx, c.primary, msg)
	if err == nil {
		c.recordLatency(time.Since(start))
		return resp, nil
	}

	if c.fallback != nil {
		resp, err = c.doExchange(ctx, *c.fallback, msg)
		if err == nil {
			c.recordLatency(time.Since(start))
			return resp, nil
		}
	}

	c.mu.Lock()
	c.metrics.FailedQueries++
	c.mu.Unlock()
	return nil, fmt.Errorf("all DoH upstreams failed: %w", err)
}

// RFC 8484: pack DNS msg to wire format, POST over HTTPS, unpack response
func (c *Client) doExchange(ctx context.Context, provider Provider, msg *dns.Msg) (*dns.Msg, error) {
	wireData, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", provider.URL, bytes.NewReader(wireData))
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("DoH request to %s failed: %w", provider.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned HTTP %d", provider.Name, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 65535))
	if err != nil {
		return nil, fmt.Errorf("read from %s failed: %w", provider.Name, err)
	}

	answer := new(dns.Msg)
	if err := answer.Unpack(body); err != nil {
		return nil, fmt.Errorf("bad response from %s: %w", provider.Name, err)
	}
	return answer, nil
}

func (c *Client) SetProvider(primary Provider) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.primary = primary
}

func (c *Client) SetFallback(fallback *Provider) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fallback = fallback
}

func (c *Client) GetMetrics() Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

func (c *Client) recordLatency(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ms := float64(d.Milliseconds())
	c.metrics.latencySum += ms
	c.metrics.latencyCount++
	c.metrics.AvgLatencyMs = c.metrics.latencySum / float64(c.metrics.latencyCount)
}

func (c *Client) ResetMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = Metrics{}
}

func (c *Client) TestConnection(ctx context.Context) error {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)
	resp, err := c.Exchange(ctx, msg)
	if err != nil {
		return fmt.Errorf("test failed: %w", err)
	}
	if resp.Rcode != dns.RcodeSuccess {
		return fmt.Errorf("unexpected rcode: %s", dns.RcodeToString[resp.Rcode])
	}
	return nil
}
