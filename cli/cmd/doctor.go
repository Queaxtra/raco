package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"raco/secretstore"
	"runtime"
)

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

func RunDoctor(ctx *Context, args []string) int {
	jsonOutput := len(args) > 0 && args[0] == "--json"
	checks := make([]doctorCheck, 0, 6)

	checks = append(checks, doctorCheck{
		Name:    "storage",
		Status:  statStatus(ctx.StoragePath),
		Details: ctx.StoragePath,
	})

	checks = append(checks, doctorCheck{
		Name:    "collections_dir",
		Status:  statStatus(filepath.Join(ctx.StoragePath, "collections")),
		Details: filepath.Join(ctx.StoragePath, "collections"),
	})

	checks = append(checks, doctorCheck{
		Name:    "environments_dir",
		Status:  statStatus(filepath.Join(ctx.StoragePath, "environments")),
		Details: filepath.Join(ctx.StoragePath, "environments"),
	})

	secretStatus := "warn"
	secretDetails := "secret backend unavailable"
	if _, err := secretstore.NewDefault(); err == nil {
		secretStatus = "ok"
		secretDetails = "secret backend available"
	}
	checks = append(checks, doctorCheck{Name: "secret_backend", Status: secretStatus, Details: secretDetails})

	checks = append(checks, doctorCheck{Name: "notifications", Status: notificationStatus(), Details: runtime.GOOS})

	if jsonOutput {
		data, _ := json.MarshalIndent(checks, "", "  ")
		fmt.Println(string(data))
		return doctorExitCode(checks)
	}

	for _, check := range checks {
		fmt.Printf("%s %s %s\n", doctorGlyph(check.Status), check.Name, check.Details)
	}
	return doctorExitCode(checks)
}

func statStatus(path string) string {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "warn"
		}
		return "fail"
	}
	return "ok"
}

func notificationStatus() string {
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("osascript"); err == nil {
			return "ok"
		}
	}
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("notify-send"); err == nil {
			return "ok"
		}
	}
	return "warn"
}

func doctorExitCode(checks []doctorCheck) int {
	for _, check := range checks {
		if check.Status == "fail" {
			return 1
		}
	}
	return 0
}

func doctorGlyph(status string) string {
	if status == "ok" {
		return "OK"
	}
	if status == "warn" {
		return "WARN"
	}
	return "FAIL"
}
