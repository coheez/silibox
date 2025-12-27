package cli

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

var (
	updateVersion string
	updateCheck   bool
	updateForce   bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install the latest sili release",
	RunE: func(cmd *cobra.Command, args []string) error {
		current := normalizeVersion(version)
		arch := normalizeArch(runtime.GOARCH)
		if arch == "" {
			return fmt.Errorf("unsupported arch: %s", runtime.GOARCH)
		}

		// Determine target version and assets
		rel, err := fetchRelease(updateVersion)
		if err != nil {
			return err
		}
		latest := normalizeVersion(rel.TagName)

		if !updateForce && !updateCheck && current != "dev" && current != "none" && cmpSemver(latest, current) <= 0 {
			fmt.Printf("sili is up to date (%s)\n", version)
			return nil
		}

		binName := fmt.Sprintf("sili-darwin-%s", arch)
		binURL := findAssetURL(rel, binName)
		if binURL == "" {
			return fmt.Errorf("could not find asset %q in release %s", binName, rel.TagName)
		}
		checksURL := findAssetURL(rel, "checksums.txt")

		if updateCheck {
			fmt.Printf("Current: %s\nLatest:  %s\nAsset:   %s\n", version, rel.TagName, binName)
			return nil
		}

		// Download binary to temp
		tmp, err := os.CreateTemp("", "sili-update-*")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())

		fmt.Printf("Downloading %s...\n", rel.TagName)
		if err := httpDownload(binURL, tmp); err != nil {
			return err
		}
		if err := tmp.Chmod(0o755); err != nil {
			return err
		}

		// Optional checksum verification
		if checksURL != "" {
			if err := verifyChecksum(checksURL, filepath.Base(binURL), tmp.Name()); err != nil {
				return fmt.Errorf("checksum verification failed: %w", err)
			}
		}

		execPath, err := os.Executable()
		if err != nil {
			return err
		}

		// Try to replace in-place
		if err := os.Rename(tmp.Name(), execPath); err != nil {
			// Likely permission denied; try sudo move
			if !errors.Is(err, os.ErrPermission) {
				// On macOS, EPERM often maps differently; fall back to sudo if rename failed for any reason
			}
			fmt.Println("Elevating privileges to install (requires sudo)...")
			if err := runSudoMove(tmp.Name(), execPath); err != nil {
				return fmt.Errorf("failed to install with sudo: %w", err)
			}
		}

		fmt.Printf("Updated sili to %s\n", rel.TagName)
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateVersion, "version", "", "Install a specific version (e.g., v0.1.1). Defaults to latest")
	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "Only check for updates; do not install")
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "Install even if the current version is newer or equal")
}

func fetchRelease(tag string) (ghRelease, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	url := "https://api.github.com/repos/coheez/silibox/releases/latest"
	if tag != "" {
		url = fmt.Sprintf("https://api.github.com/repos/coheez/silibox/releases/tags/%s", tag)
	}
	resp, err := client.Get(url)
	if err != nil {
		return ghRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ghRelease{}, fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return ghRelease{}, err
	}
	return rel, nil
}

func findAssetURL(rel ghRelease, name string) string {
	for _, a := range rel.Assets {
		if a.Name == name {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

func httpDownload(url string, w io.Writer) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

func verifyChecksum(checksURL, binaryName, filePath string) error {
	resp, err := http.Get(checksURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch checksums: %s", resp.Status)
	}

	sums := map[string]string{}
	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		line := s.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			sums[filepath.Base(parts[1])] = parts[0]
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	expected, ok := sums[binaryName]
	if !ok {
		// No matching checksum entry; skip verification
		return nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("mismatch: expected %s got %s", expected, actual)
	}
	return nil
}

func normalizeArch(goarch string) string {
	switch goarch {
	case "arm64":
		return "arm64"
	case "amd64":
		return "amd64"
	default:
		return ""
	}
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	// drop build metadata like -dirty
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v = v[:i]
	}
	return v
}

// cmpSemver compares a and b (both like 0.1.2) and returns -1, 0, 1
func cmpSemver(a, b string) int {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	for len(pa) < 3 { pa = append(pa, "0") }
	for len(pb) < 3 { pb = append(pb, "0") }
	for i := 0; i < 3; i++ {
		if pa[i] == pb[i] { continue }
		// numeric compare when possible
		// fallback to lexical
		ai, bi := atoi(pa[i]), atoi(pb[i])
		if ai < bi { return -1 }
		if ai > bi { return 1 }
		if pa[i] < pb[i] { return -1 }
		if pa[i] > pb[i] { return 1 }
	}
	return 0
}

func atoi(s string) int { n:=0; for _,c:= range s { if c<'0'||c>'9' { return n }; n = n*10 + int(c-'0') }; return n }

func runSudoMove(src, dst string) error {
	cmd := exec.Command("sudo", "mv", src, dst)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}