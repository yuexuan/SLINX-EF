package update

import (
	"encoding/json"
 "crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
 "time"
	"os/exec"

	"runtime"
	"strings"

	"github.com/slinxlink/node/internal/util"
)

type CheckResult struct {
	HasUpdate     bool   `json:"has_update"`
	LatestVersion string `json:"latest_version"`
	Changelog     string `json:"changelog"`
}

type releaseInfo struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
}

func getRepo() string { return "https://github.com/yuexuan/SLINX-EF" }

func apiURL() string {
	path := strings.TrimPrefix(getRepo(), "https://github.com/")
	return "https://api.github.com/repos/" + path + "/releases/latest"
}

func downloadURL(version string) string {
	return fmt.Sprintf("%s/releases/download/%s/slinx_linux_%s", getRepo(), version, runtime.GOARCH)
}

func Check(currentVersion string) (*CheckResult, error) {
	resp, err := http.Get(apiURL())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var release releaseInfo
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, err
	}

	return &CheckResult{
		HasUpdate:     release.TagName != "" && release.TagName != currentVersion,
		LatestVersion: release.TagName,
		Changelog:     release.Body,
	}, nil
}

func Update(currentVersion string) error {
	result, err := Check(currentVersion)
	if err != nil {
		util.Error("[update] 版本检测失败: %v", err)
		return err
	}
	if !result.HasUpdate {
		return nil
	}

	util.Info("[update] 开始更新: %s → %s", currentVersion, result.LatestVersion)

	exe, err := os.Executable()
	if err != nil {
		return err
	}

 script, err := os.CreateTemp("", "slinx-ef-update-*.sh")
 if err != nil { return err }
 resp, err := http.Get(getRepo()+"/releases/download/"+result.LatestVersion+"/install.sh")
 if err != nil { return err }; defer resp.Body.Close()
 if resp.StatusCode != 200 { return fmt.Errorf("installer HTTP %d", resp.StatusCode) }
 _, err = io.Copy(script, resp.Body); script.Close(); if err != nil { return err }
 defer func(){ if err != nil { os.Remove(script.Name()) } }()
 sums, err := http.Get(getRepo()+"/releases/download/"+result.LatestVersion+"/SHA256SUMS")
 if err != nil { return err }; defer sums.Body.Close()
 if sums.StatusCode != 200 { return fmt.Errorf("checksum HTTP %d", sums.StatusCode) }
 sumData, err := io.ReadAll(sums.Body); if err != nil { return err }
 content, err := os.ReadFile(script.Name()); if err != nil { return err }
 expected := fmt.Sprintf("%x  install.sh", sha256.Sum256(content))
 found := false; for _, line := range strings.Split(string(sumData), "\n") { if strings.TrimSpace(line)==expected { found=true } }
 if !found { return fmt.Errorf("installer checksum mismatch") }
 // Launch outside the service cgroup so restart does not kill the installer.
 unit := fmt.Sprintf("slinx-ef-update-%d", time.Now().Unix())
 return exec.Command("systemd-run", "--unit="+unit, "/bin/bash", script.Name(), result.LatestVersion).Run()
}
