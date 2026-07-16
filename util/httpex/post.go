package httpex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// defaultClient is a shared HTTP client with connection pooling enabled.
// Creating a new http.Client per request disables keep-alive and connection
// reuse, which hurts latency and throughput. This client uses the default
// transport with sensible pool sizes.
var defaultClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	},
}

// Post sends a POST request with form-encoded parameters.
// For context-aware usage, prefer PostCtx.
func Post(ul string, params *url.Values, timeout time.Duration, hds ...http.Header) (*http.Response, error) {
	return PostCtx(context.Background(), ul, params, timeout, hds...)
}

// PostCtx sends a POST request with form-encoded parameters and context support.
// The context is used for request cancellation and deadlines.
func PostCtx(ctx context.Context, ul string, params *url.Values, timeout time.Duration, hds ...http.Header) (*http.Response, error) {
	if params == nil {
		params = &url.Values{}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, ul, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating POST request: %w", err)
	}
	header := http.Header{}
	if len(hds) > 0 {
		header = hds[0]
	}
	header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	request.Header = header
	client := *defaultClient
	client.Timeout = time.Second * timeout
	res, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("executing POST request: %w", err)
	}
	return res, nil
}

// Posts sends a POST request and returns the status code and response body.
// For context-aware usage, prefer PostsCtx.
func Posts(ul string, params *url.Values, timeout time.Duration, hds ...http.Header) (int, []byte, error) {
	return PostsCtx(context.Background(), ul, params, timeout, hds...)
}

// PostsCtx sends a POST request with context support and returns the status code and response body.
func PostsCtx(ctx context.Context, ul string, params *url.Values, timeout time.Duration, hds ...http.Header) (int, []byte, error) {
	if params == nil {
		params = &url.Values{}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, ul, strings.NewReader(params.Encode()))
	if err != nil {
		return 0, nil, fmt.Errorf("creating POST request: %w", err)
	}
	header := http.Header{}
	if len(hds) > 0 {
		header = hds[0]
	}
	header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	request.Header = header
	client := *defaultClient
	client.Timeout = time.Second * timeout
	res, err := client.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("executing POST request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	bts, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("reading response body: %w", err)
	}
	return res.StatusCode, bts, nil
}

// PostJSON sends a POST request with a JSON body.
// For context-aware usage, prefer PostJSONCtx.
func PostJSON(ul string, params any, timeout time.Duration, hds ...http.Header) (*http.Response, error) {
	return PostJSONCtx(context.Background(), ul, params, timeout, hds...)
}

// PostJSONCtx sends a POST request with a JSON body and context support.
func PostJSONCtx(ctx context.Context, ul string, params any, timeout time.Duration, hds ...http.Header) (*http.Response, error) {
	js, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON body: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, ul, bytes.NewReader(js))
	if err != nil {
		return nil, fmt.Errorf("creating POST JSON request: %w", err)
	}
	header := http.Header{}
	if len(hds) > 0 {
		header = hds[0]
	}
	header.Add("Content-Type", "application/json; charset=utf-8")
	request.Header = header
	client := *defaultClient
	client.Timeout = time.Second * timeout
	res, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("executing POST JSON request: %w", err)
	}
	return res, nil
}

// PostResult sends a POST request and unmarshals the JSON response into result.
// For context-aware usage, prefer PostResultCtx.
func PostResult(ul string, params *url.Values, result any, timeout time.Duration, hds ...http.Header) (int, []byte, error) {
	return PostResultCtx(context.Background(), ul, params, result, timeout, hds...)
}

// PostResultCtx sends a POST request with context support and unmarshals the JSON response into result.
func PostResultCtx(ctx context.Context, ul string, params *url.Values, result any, timeout time.Duration, hds ...http.Header) (int, []byte, error) {
	if result == nil {
		return 0, nil, errors.New("result is nil")
	}
	res, err := PostCtx(ctx, ul, params, timeout, hds...)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = res.Body.Close() }()
	bts, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, fmt.Errorf("reading response body: %w", err)
	}
	if res.StatusCode != 200 {
		return res.StatusCode, bts, fmt.Errorf("response error (code:%d): %s", res.StatusCode, string(bts))
	}
	if err := json.Unmarshal(bts, result); err != nil {
		return res.StatusCode, bts, fmt.Errorf("unmarshaling response: %w", err)
	}
	return res.StatusCode, bts, nil
}

// PostJSONResult sends a POST request with a JSON body and unmarshals the response into result.
// For context-aware usage, prefer PostJSONResultCtx.
func PostJSONResult(ul string, params any, result any, timeout time.Duration, hds ...http.Header) (int, []byte, error) {
	return PostJSONResultCtx(context.Background(), ul, params, result, timeout, hds...)
}

// PostJSONResultCtx sends a POST request with a JSON body, context support, and unmarshals the response into result.
func PostJSONResultCtx(ctx context.Context, ul string, params any, result any, timeout time.Duration, hds ...http.Header) (int, []byte, error) {
	if result == nil {
		return 0, nil, errors.New("result is nil")
	}
	res, err := PostJSONCtx(ctx, ul, params, timeout, hds...)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = res.Body.Close() }()
	bts, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, fmt.Errorf("reading response body: %w", err)
	}
	if res.StatusCode != 200 {
		return res.StatusCode, bts, fmt.Errorf("response error (code:%d): %s", res.StatusCode, string(bts))
	}
	if err := json.Unmarshal(bts, result); err != nil {
		return res.StatusCode, bts, fmt.Errorf("unmarshaling response: %w", err)
	}
	return res.StatusCode, bts, nil
}
