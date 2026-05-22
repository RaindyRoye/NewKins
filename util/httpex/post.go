package httpex

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func Post(ul string, params *url.Values, timeout time.Duration, hds ...http.Header) (*http.Response, error) {
	if params == nil {
		params = &url.Values{}
	}
	request, err := http.NewRequest(http.MethodPost, ul, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	header := http.Header{}
	if len(hds) > 0 {
		header = hds[0]
	}
	header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	request.Header = header
	client := &http.Client{}
	client.Timeout = time.Second * timeout
	return client.Do(request)
}
func Posts(ul string, params *url.Values, timeout time.Duration, hds ...http.Header) (int, []byte, error) {
	if params == nil {
		params = &url.Values{}
	}
	request, err := http.NewRequest(http.MethodPost, ul, strings.NewReader(params.Encode()))
	if err != nil {
		return 0, nil, err
	}
	header := http.Header{}
	if len(hds) > 0 {
		header = hds[0]
	}
	header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	request.Header = header
	client := &http.Client{}
	client.Timeout = time.Second * timeout
	res, err := client.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()

	bts, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("reading response body: %w", err)
	}
	return res.StatusCode, bts, nil
}
func PostJSON(ul string, params interface{}, timeout time.Duration, hds ...http.Header) (*http.Response, error) {
	js, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest(http.MethodPost, ul, bytes.NewReader(js))
	if err != nil {
		return nil, err
	}
	header := http.Header{}
	if len(hds) > 0 {
		header = hds[0]
	}
	header.Add("Content-Type", "application/json; charset=utf-8")
	request.Header = header
	client := &http.Client{}
	client.Timeout = time.Second * timeout
	return client.Do(request)
}

func PostResult(ul string, params *url.Values, result interface{}, timeout time.Duration, hds ...http.Header) (int, []byte, error) {
	if result == nil {
		return 0, nil, errors.New("result is nil")
	}
	res, err := Post(ul, params, timeout, hds...)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	bts, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, fmt.Errorf("reading response body: %w", err)
	}
	if res.StatusCode != 200 {
		return res.StatusCode, bts, fmt.Errorf("response err(code:%d): %s", res.StatusCode, string(bts))
	}
	return res.StatusCode, bts, json.Unmarshal(bts, result)
}
func PostJSONResult(ul string, params interface{}, result interface{}, timeout time.Duration, hds ...http.Header) (int, []byte, error) {
	if result == nil {
		return 0, nil, errors.New("result is nil")
	}
	res, err := PostJSON(ul, params, timeout, hds...)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	bts, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, fmt.Errorf("reading response body: %w", err)
	}
	if res.StatusCode != 200 {
		return res.StatusCode, bts, fmt.Errorf("response err(code:%d): %s", res.StatusCode, string(bts))
	}
	return res.StatusCode, bts, json.Unmarshal(bts, result)
}
