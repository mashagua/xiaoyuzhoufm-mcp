package constants

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestAppMetadataUsesEnvironment(t *testing.T) {
	if os.Getenv("TEST_CONSTANTS_ENV") == "1" {
		if XiaoyuzhouAppVersion != "9.8.7" {
			t.Fatalf("XiaoyuzhouAppVersion = %q, want %q", XiaoyuzhouAppVersion, "9.8.7")
		}
		if XiaoyuzhouAppBuildNo != "987654" {
			t.Fatalf("XiaoyuzhouAppBuildNo = %q, want %q", XiaoyuzhouAppBuildNo, "987654")
		}
		if !strings.Contains(XiaoyuzhouUserAgent, "Xiaoyuzhou/9.8.7 (build:987654;") {
			t.Fatalf("XiaoyuzhouUserAgent = %q", XiaoyuzhouUserAgent)
		}
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestAppMetadataUsesEnvironment")
	cmd.Env = append(os.Environ(),
		"TEST_CONSTANTS_ENV=1",
		"XIAOYUZHOU_APP_VERSION=9.8.7",
		"XIAOYUZHOU_APP_BUILD_NO=987654",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("subprocess failed: %v\n%s", err, output)
	}
}
