package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"raco/model"
	"raco/util"
	"strings"
	"time"
)

// privateNets parsed once to avoid repeated ParseCIDR on every request.
var privateNets []*net.IPNet

func init() {
	cidrs := []string{
		"127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"169.254.0.0/16", "::1/128", "fe80::/10", "fc00::/7",
	}
	privateNets = make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		privateNets = append(privateNets, n)
	}
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           safeDialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
	}

	return &Client{
		httpClient: &http.Client{
			Timeout:       5 * time.Minute,
			Transport:     transport,
			CheckRedirect: safeRedirectCheck,
		},
	}
}

const (
	defaultRequestTimeout = 30 * time.Second
	maxRetries            = 3
	retryBaseDelay        = 1 * time.Second
)

func requestTimeout(req *model.Request) time.Duration {
	if req != nil && req.TimeoutSeconds > 0 {
		t := time.Duration(req.TimeoutSeconds) * time.Second
		if t > 5*time.Minute {
			return 5 * time.Minute
		}
		return t
	}
	return defaultRequestTimeout
}

func isIdempotentMethod(method string) bool {
	switch method {
	case "GET", "HEAD", "PUT", "DELETE":
		return true
	}
	return false
}

func isRetryableStatus(code int) bool {
	if code >= 500 {
		return true
	}
	if code == 429 {
		return true
	}
	return false
}

func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if isPrivateIP(ip) {
			return nil, errors.New("connection to private IP blocked")
		}
	}

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	return dialer.DialContext(ctx, network, addr)
}

func isPrivateIP(ip net.IP) bool {
	for _, n := range privateNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func safeRedirectCheck(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("too many redirects")
	}

	if !util.ValidateURL(req.URL.String()) {
		return errors.New("redirect to invalid URL blocked")
	}

	return nil
}

func (c *Client) Execute(req *model.Request) (*model.Response, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}

	if !util.ValidateURL(req.URL) {
		return nil, errors.New("invalid URL")
	}

	if !util.ValidateMethod(req.Method) {
		return nil, errors.New("invalid HTTP method")
	}

	timeout := requestTimeout(req)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var lastResp *model.Response
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * (1 << (attempt - 1))
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			select {
			case <-ctx.Done():
				if lastErr != nil {
					return nil, lastErr
				}
				return lastResp, nil
			case <-time.After(delay):
			}
		}

		httpReq, err := c.buildRequest(req)
		if err != nil {
			return nil, err
		}
		httpReq = httpReq.WithContext(ctx)

		startTime := time.Now()
		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return nil, err
			}
			continue
		}

		maxBodySize := int64(10 * 1024 * 1024)
		limitedReader := io.LimitReader(httpResp.Body, maxBodySize)
		body, readErr := io.ReadAll(limitedReader)
		httpResp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		headerLen := len(httpResp.Header)
		headers := make(map[string]string, headerLen)
		for key, values := range httpResp.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}

		resp := &model.Response{
			StatusCode: httpResp.StatusCode,
			Headers:    headers,
			Body:       string(body),
			Duration:   time.Since(startTime),
			Timestamp:  time.Now(),
		}
		lastResp = resp
		lastErr = nil

		shouldRetry := isIdempotentMethod(req.Method) && isRetryableStatus(httpResp.StatusCode) && attempt < maxRetries
		if !shouldRetry {
			return resp, nil
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return lastResp, nil
}

func (c *Client) buildRequest(req *model.Request) (*http.Request, error) {
	var bodyReader io.Reader
	var contentType string

	if len(req.Files) > 0 {
		body, ct, err := buildMultipartBody(req)
		if err != nil {
			return nil, err
		}
		bodyReader = body
		contentType = ct
	}

	if req.Body != "" && len(req.Files) == 0 {
		bodyReader = strings.NewReader(req.Body)
	}

	requestURL := req.URL
	if len(req.Query) > 0 {
		parsed, err := url.Parse(req.URL)
		if err != nil {
			return nil, err
		}
		q := parsed.Query()
		for k, v := range req.Query {
			q.Set(k, v)
		}
		parsed.RawQuery = q.Encode()
		requestURL = parsed.String()
	}

	httpReq, err := http.NewRequest(req.Method, requestURL, bodyReader)
	if err != nil {
		return nil, err
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	return httpReq, nil
}

func buildMultipartBody(req *model.Request) (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for _, file := range req.Files {
		if err := file.Validate(); err != nil {
			writer.Close()
			return nil, "", err
		}

		fileData, err := file.ReadData()
		if err != nil {
			writer.Close()
			return nil, "", err
		}

		part, err := writer.CreateFormFile(file.FieldName, file.FileName)
		if err != nil {
			writer.Close()
			return nil, "", err
		}

		if _, err := part.Write(fileData); err != nil {
			writer.Close()
			return nil, "", err
		}
	}

	if req.Body != "" {
		bodyMap := make(map[string]string)
		pairs := strings.Split(req.Body, "&")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				bodyMap[parts[0]] = parts[1]
			}
		}
		for key, value := range bodyMap {
			if err := writer.WriteField(key, value); err != nil {
				writer.Close()
				return nil, "", err
			}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return &buf, writer.FormDataContentType(), nil
}

func SaveDownloadedFile(resp *model.Response, downloadPath string) (*model.FileDownload, error) {
	// Resolve the canonical path before creating any directories to prevent path traversal.
	cleanPath := filepath.Clean(downloadPath)
	dir := filepath.Dir(cleanPath)

	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, err
	}

	// Re-evaluate after MkdirAll so EvalSymlinks can resolve the full path.
	absDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, err
	}
	finalPath := filepath.Join(absDir, filepath.Base(cleanPath))

	file, err := os.OpenFile(finalPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err := file.WriteString(resp.Body); err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	contentType := resp.Headers["Content-Type"]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return &model.FileDownload{
		FilePath:     finalPath,
		OriginalName: filepath.Base(finalPath),
		ContentType:  contentType,
		Size:         info.Size(),
	}, nil
}

func ReplaceEnvVars(input string, env *model.Environment) string {
	if env == nil || len(env.Variables) == 0 {
		return input
	}

	result := input
	for key, value := range env.Variables {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

func ReplaceEnvVarsInMap(m map[string]string, env *model.Environment) map[string]string {
	if env == nil || len(m) == 0 {
		return m
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = ReplaceEnvVars(v, env)
	}
	return out
}
