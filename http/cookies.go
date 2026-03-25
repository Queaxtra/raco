package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"raco/util"
	"regexp"
	"strings"
)

type persistedCookies struct {
	URL     string         `json:"url"`
	Cookies []*http.Cookie `json:"cookies"`
}

// Cookie jar names are intentionally narrow so persisted files stay predictable
// and cannot be abused as arbitrary file paths.
var cookieJarNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// loadPersistentJar restores only bounded, validated cookie files from the
// managed cookie directory under the user's Raco home.
func loadPersistentJar(path string, targetURL string) (*cookiejar.Jar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return jar, nil
	}
	resolvedPath, err := resolveCookieJarFile(path)
	if err != nil {
		return nil, err
	}
	data, err := util.ReadFileBounded(resolvedPath, 512*1024)
	if err != nil {
		if os.IsNotExist(err) {
			return jar, nil
		}
		return nil, err
	}
	var persisted persistedCookies
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, err
	}
	rawURL := persisted.URL
	if rawURL == "" {
		rawURL = targetURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	jar.SetCookies(parsed, persisted.Cookies)
	return jar, nil
}

// savePersistentJar writes cookies atomically so interrupted runs never leave a
// partially written jar that corrupts later sessions.
func savePersistentJar(path string, rawURL string, jar *cookiejar.Jar) error {
	if path == "" || jar == nil {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(persistedCookies{
		URL:     rawURL,
		Cookies: jar.Cookies(parsed),
	}, "", "  ")
	if err != nil {
		return err
	}
	cleanPath, err := resolveCookieJarFile(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
		return err
	}
	tempPath := cleanPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return os.Rename(tempPath, cleanPath)
}

// resolveCookieJarFile normalizes every caller onto the same canonical cookie
// storage root. The CLI may accept loose names, but persistence never does.
func resolveCookieJarFile(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", fmt.Errorf("cookie jar name is required")
	}
	name = filepath.Base(name)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if !cookieJarNamePattern.MatchString(name) {
		return "", fmt.Errorf("cookie jar name is invalid")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	baseDir := filepath.Join(homeDir, ".raco", "cookies")
	return util.ResolveContainedPath(baseDir, name+".json")
}
