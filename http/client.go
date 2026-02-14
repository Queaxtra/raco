package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"raco/model"
	"raco/util"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	transport := &http.Transport{
		Proxy: nil,
		DialContext: safeDialContext,
		ForceAttemptHTTP2: true,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &Client{
		httpClient: &http.Client{
			Timeout:       30 * time.Second,
			Transport:     transport,
			CheckRedirect: safeRedirectCheck,
		},
	}
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
	privateCIDRs := []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"::1/128",
		"fe80::/10",
		"fc00::/7",
	}

	for _, cidr := range privateCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
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

	startTime := time.Now()

	httpReq, err := c.buildRequest(req)
	if err != nil {
		return nil, err
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	maxBodySize := int64(10 * 1024 * 1024)
	limitedReader := io.LimitReader(httpResp.Body, maxBodySize)

	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string)
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

	return resp, nil
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
		bodyReader = bytes.NewBufferString(req.Body)
	}

	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
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
	dir := filepath.Dir(downloadPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	file, err := os.Create(downloadPath)
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
		FilePath:     downloadPath,
		OriginalName: filepath.Base(downloadPath),
		ContentType:  contentType,
		Size:         info.Size(),
	}, nil
}

func ReplaceEnvVars(input string, env *model.Environment) string {
	if env == nil {
		return input
	}

	result := input
	for key, value := range env.Variables {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}
