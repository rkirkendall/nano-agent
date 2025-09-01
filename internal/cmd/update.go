package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/rkirkendall/nano-agent/internal/version"
)

const repoOwner = "rkirkendall"
const repoName = "nano-agent"

func latestVersionTag() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "nano-agent-updater")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	type obj struct {
		Tag string `json:"tag_name"`
	}
	var o obj
	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		return "", err
	}
	return strings.TrimSpace(o.Tag), nil
}

// maybeSelfUpdate checks GitHub for a newer release and prints an inline hint.
// For a conservative first version, we do not auto-replace the binary; we only suggest the one-liner.
func maybeSelfUpdate(out interface{ Println(a ...any) }) {
	// Skip in dev builds
	if version.Version == "dev" {
		return
	}
	tag, err := latestVersionTag()
	if err != nil || tag == "" {
		return
	}
	if tag == version.Version {
		return
	}

	// Suggest an install one-liner based on OS
	var hint string
	switch runtime.GOOS {
	case "darwin", "linux":
		hint = fmt.Sprintf("curl -fsSL https://raw.githubusercontent.com/%s/%s/main/scripts/install.sh | bash", repoOwner, repoName)
	case "windows":
		hint = fmt.Sprintf("powershell -ExecutionPolicy Bypass -c \"iwr https://raw.githubusercontent.com/%s/%s/main/scripts/install.ps1 -UseB | iex\"", repoOwner, repoName)
	}
	if hint != "" {
		out.Println("A newer nano-agent is available (", tag, ") — update with:")
		out.Println("  ", hint)
	}
}
