package cmd

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	updateRepo        = "Queaxtra/raco"
	updateAPILatest   = "https://api.github.com/repos/" + updateRepo + "/releases/latest"
	updateDownloadURL = "https://github.com/" + updateRepo + "/releases/download/%s/raco_%s_%s.tar.gz"
	updateChecksumURL = "https://github.com/" + updateRepo + "/releases/download/%s/checksums.txt"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func RunUpdate() int {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, updateAPILatest, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching latest release: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: GitHub API returned %d\n", resp.StatusCode)
		return 1
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing release: %v\n", err)
		return 1
	}

	tag := strings.TrimSpace(release.TagName)
	if tag == "" {
		fmt.Fprintf(os.Stderr, "Error: no tag_name in release\n")
		return 1
	}

	gos := runtime.GOOS
	if gos == "darwin" {
		gos = "darwin"
	}
	if gos != "darwin" && gos != "linux" {
		fmt.Fprintf(os.Stderr, "Error: update is only supported on darwin and linux\n")
		return 1
	}

	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" && arch != "aarch64" {
		fmt.Fprintf(os.Stderr, "Error: unsupported arch %s\n", arch)
		return 1
	}
	if arch == "aarch64" {
		arch = "arm64"
	}

	tarballURL := fmt.Sprintf(updateDownloadURL, tag, gos, arch)
	checksumURL := fmt.Sprintf(updateChecksumURL, tag)

	fmt.Printf("Latest release: %s\n", tag)
	fmt.Printf("Downloading raco_%s_%s.tar.gz...\n", gos, arch)

	tarballPath, err := downloadFile(client, tarballURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	defer os.Remove(tarballPath)

	expectedSum, err := fetchChecksum(client, checksumURL, fmt.Sprintf("raco_%s_%s.tar.gz", gos, arch))
	if err == nil {
		if err := verifyChecksum(tarballPath, expectedSum); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
	}

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	newPath := exePath + ".new"
	if err := extractBinary(tarballPath, newPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	defer os.Remove(newPath)

	if err := os.Chmod(newPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if err := os.Rename(newPath, exePath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot replace binary (try running with sudo): %v\n", err)
		return 1
	}

	fmt.Printf("Updated to %s. Run 'raco --version' to verify.\n", tag)
	return 0
}

func downloadFile(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %d", resp.StatusCode)
	}

	f, err := os.CreateTemp("", "raco_*.tar.gz")
	if err != nil {
		return "", err
	}
	path := f.Name()
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

func fetchChecksum(client *http.Client, url, filename string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && (parts[1] == filename || strings.HasSuffix(parts[1], filename)) {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum not found for %s", filename)
}

func verifyChecksum(path, expectedHex string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	expected := strings.TrimSpace(strings.ToLower(expectedHex))
	actual = strings.ToLower(actual)

	if actual != expected {
		return fmt.Errorf("checksum mismatch (expected %s, got %s)", expected, actual)
	}
	return nil
}

func extractBinary(tarballPath, destPath string) error {
	f, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) != "raco" {
			continue
		}

		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, tr)
		out.Close()
		return err
	}
	return fmt.Errorf("raco binary not found in archive")
}
