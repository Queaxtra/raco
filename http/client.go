package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
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

// NewClient builds the default hardened transport once so startup fails fast if
// a future transport default becomes invalid.
func NewClient() *Client {
	transport, err := defaultTransport(false, model.TransportConfig{})
	if err != nil {
		panic(err)
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
	return safeDialContextWithConfig(false)(ctx, network, addr)
}

func safeDialContextWithConfig(allowPrivate bool) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network string, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}

		ip := net.ParseIP(host)
		if ip != nil {
			if isPrivateIP(ip) && !allowPrivate {
				return nil, errors.New("connection to private IP blocked")
			}
		}
		if ip == nil && !allowPrivate {
			// DNS rebinding defenses must validate resolved IPs, not only the raw hostname.
			ips, lookupErr := net.DefaultResolver.LookupIPAddr(ctx, host)
			if lookupErr != nil {
				return nil, lookupErr
			}
			for _, candidate := range ips {
				if isPrivateIP(candidate.IP) {
					return nil, errors.New("connection to private IP blocked")
				}
			}
		}

		dialer := &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}

		return dialer.DialContext(ctx, network, addr)
	}
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

	httpClient, jar, err := c.httpClientForRequest(req)
	if err != nil {
		return nil, err
	}

	rateLimitDelay(req)

	var lastResp *model.Response
	var lastErr error

	maxAttempts := retryAttempts(req)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := retryDelay(req, attempt)
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
		httpResp, err := httpClient.Do(httpReq)
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

		shouldRetry := shouldRetryRequest(req, httpResp.StatusCode, attempt, maxAttempts)
		if !shouldRetry {
			_ = saveCookieJar(req, requestURLForPersistence(httpReq), jar)
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

func (c *Client) httpClientForRequest(req *model.Request) (*http.Client, *cookiejar.Jar, error) {
	jar, err := loadPersistentJar(req.CookieJar, req.URL)
	if err != nil {
		return nil, nil, err
	}
	// Transport validation happens per request so invalid proxy or TLS settings
	// are rejected before any outbound connection is attempted.
	transport, err := defaultTransport(req.Transport.AllowPrivateIP, req.Transport)
	if err != nil {
		return nil, nil, err
	}

	return &http.Client{
		Timeout:       5 * time.Minute,
		Transport:     transport,
		CheckRedirect: safeRedirectCheck,
		Jar:           jar,
	}, jar, nil
}

// defaultTransport centralizes all transport hardening so HTTP execution,
// collection runs, and future protocol adapters share one validation path.
func defaultTransport(allowPrivate bool, transportCfg model.TransportConfig) (*http.Transport, error) {
	if err := validateTransportConfig(transportCfg); err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy:               nil,
		DialContext:         safeDialContextWithConfig(allowPrivate),
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 16,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if transportCfg.CAFile != "" {
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			return nil, fmt.Errorf("failed to load system cert pool")
		}
		// CA bundles are loaded explicitly instead of silently ignored so broken
		// trust configuration fails during setup rather than during incident response.
		data, readErr := os.ReadFile(filepath.Clean(transportCfg.CAFile))
		if readErr != nil {
			return nil, readErr
		}
		if ok := pool.AppendCertsFromPEM(data); !ok {
			return nil, fmt.Errorf("invalid CA bundle")
		}
		tlsConfig.RootCAs = pool
	}
	if transportCfg.CertFile != "" && transportCfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(filepath.Clean(transportCfg.CertFile), filepath.Clean(transportCfg.KeyFile))
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	transport.TLSClientConfig = tlsConfig
	if transportCfg.ProxyURL != "" {
		parsed, err := url.Parse(transportCfg.ProxyURL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(parsed)
	}
	return transport, nil
}

// validateTransportConfig keeps the CLI surface strict and predictable. We do
// not allow insecure TLS or ambiguous client certificate configuration in prod.
func validateTransportConfig(transportCfg model.TransportConfig) error {
	if transportCfg.AllowInsecure {
		return fmt.Errorf("insecure TLS is disabled")
	}
	if transportCfg.RateLimitPerMin < 0 {
		return fmt.Errorf("rate limit must be positive")
	}
	if transportCfg.RateLimitPerMin > 60000 {
		return fmt.Errorf("rate limit exceeds maximum")
	}
	if transportCfg.ProxyURL != "" {
		// Proxy credentials in URLs are rejected because they are easy to leak via
		// shell history, process listings, and error output.
		parsed, err := url.Parse(transportCfg.ProxyURL)
		if err != nil {
			return fmt.Errorf("invalid proxy URL: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("proxy scheme must be http or https")
		}
		if parsed.Host == "" {
			return fmt.Errorf("proxy host is required")
		}
		if parsed.User != nil {
			return fmt.Errorf("proxy credentials in URL are not allowed")
		}
	}
	if (transportCfg.CertFile == "") != (transportCfg.KeyFile == "") {
		return fmt.Errorf("client certificate and key must be provided together")
	}
	for _, candidate := range []string{transportCfg.CAFile, transportCfg.CertFile, transportCfg.KeyFile} {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(filepath.Clean(candidate))
		if err != nil {
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("transport file path points to a directory")
		}
	}
	return nil
}

func retryAttempts(req *model.Request) int {
	if req == nil || req.Retry.MaxAttempts <= 0 {
		return maxRetries + 1
	}
	if req.Retry.MaxAttempts > 10 {
		return 10
	}
	return req.Retry.MaxAttempts
}

func retryDelay(req *model.Request, attempt int) time.Duration {
	base := retryBaseDelay
	if req != nil && req.Retry.BaseDelayMS > 0 {
		base = time.Duration(req.Retry.BaseDelayMS) * time.Millisecond
	}
	delay := base * (1 << (attempt - 1))
	maxDelay := 30 * time.Second
	if req != nil && req.Retry.MaxDelayMS > 0 {
		maxDelay = time.Duration(req.Retry.MaxDelayMS) * time.Millisecond
	}
	if delay > maxDelay {
		delay = maxDelay
	}
	jitter := time.Duration(time.Now().UnixNano()%int64(base+time.Millisecond)) / 2
	return delay + jitter
}

func shouldRetryRequest(req *model.Request, statusCode int, attempt int, maxAttempts int) bool {
	if attempt >= maxAttempts-1 {
		return false
	}
	if !isRetryableStatus(statusCode) {
		return false
	}
	if req != nil && req.Retry.RetryNonIdempotent {
		return true
	}
	return isIdempotentMethod(req.Method)
}

func rateLimitDelay(req *model.Request) {
	if req == nil || req.Transport.RateLimitPerMin <= 0 {
		return
	}
	delay := time.Minute / time.Duration(req.Transport.RateLimitPerMin)
	if delay <= 0 {
		return
	}
	time.Sleep(delay)
}

func requestURLForPersistence(req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}
	return req.URL.String()
}

func saveCookieJar(req *model.Request, rawURL string, jar *cookiejar.Jar) error {
	if req == nil || req.CookieJar == "" {
		return nil
	}
	return savePersistentJar(req.CookieJar, rawURL, jar)
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

func ReplaceEnvVars(input string, env VariableProvider) string {
	if env == nil {
		return input
	}
	variables := env.GetVariables()
	if len(variables) == 0 {
		return input
	}

	result := input
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

func ReplaceEnvVarsInMap(m map[string]string, env VariableProvider) map[string]string {
	if env == nil || len(m) == 0 {
		return m
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = ReplaceEnvVars(v, env)
	}
	return out
}
