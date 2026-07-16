package ci

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPagesWorkflowKeepsSecurityGatesBeforeArtifactAndDeploy(t *testing.T) {
	t.Parallel()
	filename := filepath.Join("..", "..", ".github", "workflows", "pages.yaml")
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatalf("invalid workflow YAML: %v", err)
	}
	text := string(data)
	required := []string{
		"workflow_dispatch:",
		"source_ref:",
		"BUILD_MODE=legacy",
		"BUILD_MODE=hextra",
		"scripts/prepare-legacy-pages.sh",
		"contents: read",
		"pages: write",
		"id-token: write",
		"persist-credentials: false",
		"cancel-in-progress: false",
		"go test -race -count=1 ./...",
		"contentctl verify",
		"--redact=100",
		"hugo --gc --minify",
		"linkcheck public",
		"actions/upload-pages-artifact@56afc609e74202658d3ffba0e8f6dda462b719fa",
		"needs: build",
		"if: ${{ inputs.deploy == true }}",
		"actions/deploy-pages@d6db90164ac5ed86f2b6aed7e0febac5b3c0c03e",
	}
	for _, value := range required {
		if !strings.Contains(text, value) {
			t.Fatalf("workflow is missing %q", value)
		}
	}
	if strings.Contains(text, "\n  push:") {
		t.Fatal("automatic main push deployment must remain disabled until preview confirmation")
	}
	ordered := []string{
		"go test -race -count=1 ./...",
		"contentctl verify",
		"Run independent redacted tracked-tree Gitleaks check",
		"Build production site",
		"Check generated internal links",
		"actions/upload-pages-artifact@56afc609e74202658d3ffba0e8f6dda462b719fa",
	}
	previous := -1
	for _, marker := range ordered {
		index := strings.Index(text, marker)
		if index <= previous {
			t.Fatalf("workflow gate %q is absent or out of order", marker)
		}
		previous = index
	}
	for _, forbidden := range []string{"node", "yarn", "vuepress", "force push", "push --force"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("workflow contains forbidden legacy tool or publish path %q", forbidden)
		}
	}
	for _, mutable := range []string{"actions/checkout@v", "actions/setup-go@v", "actions/upload-pages-artifact@v", "actions/deploy-pages@v"} {
		if strings.Contains(text, mutable) {
			t.Fatalf("workflow contains mutable action reference %q", mutable)
		}
	}
	buildStart := strings.Index(text, "\n  build:")
	deployStart := strings.Index(text, "\n  deploy:")
	if buildStart < 0 || deployStart <= buildStart {
		t.Fatal("workflow jobs are malformed")
	}
	buildText := text[buildStart:deployStart]
	if strings.Contains(buildText, "pages: write") || strings.Contains(buildText, "id-token: write") {
		t.Fatal("build job has deployment permissions")
	}
}

func TestVersionSingleSourcesStayAligned(t *testing.T) {
	t.Parallel()
	configFile, err := os.Open(filepath.Join("..", "..", "config", "versions.env"))
	if err != nil {
		t.Fatal(err)
	}
	defer configFile.Close()
	versions := make(map[string]string)
	reader := bufio.NewScanner(configFile)
	for reader.Scan() {
		key, value, ok := strings.Cut(reader.Text(), "=")
		if ok {
			versions[key] = value
		}
	}
	if err := reader.Err(); err != nil {
		t.Fatal(err)
	}
	moduleBytes, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	moduleText := string(moduleBytes)
	if !strings.Contains(moduleText, "\ngo "+versions["GO_VERSION"]+"\n") {
		t.Fatal("go.mod Go version differs from config/versions.env")
	}
	if !strings.Contains(moduleText, "github.com/imfing/hextra "+versions["HEXTRA_VERSION"]) {
		t.Fatal("go.mod Hextra version differs from config/versions.env")
	}
	for _, key := range []string{
		"HUGO_LINUX_AMD64_SHA256",
		"GITLEAKS_LINUX_AMD64_ARCHIVE_SHA256",
		"GITLEAKS_LINUX_AMD64_BINARY_SHA256",
		"FLEXSEARCH_SHA256",
		"MERMAID_SHA256",
	} {
		if len(versions[key]) != 64 {
			t.Fatalf("%s is not a pinned full digest", key)
		}
	}
	if len(versions["LEGACY_STATIC_COMMIT"]) != 40 {
		t.Fatal("legacy recovery commit is not a full commit SHA")
	}
	if versions["LEGACY_STATIC_COMMIT"] != "400aaafaa932a814585497cf85685d9b6516756b" {
		t.Fatal("legacy recovery commit changed without updating recovery evidence")
	}
}

func TestLegacyRecoveryUsesPinnedStaticTreeAndIntegrityChecks(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("..", "..", "scripts", "prepare-legacy-pages.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{
		"LEGACY_STATIC_COMMIT",
		"git merge-base --is-ancestor",
		"git archive --format=tar",
		"git ls-tree -rz",
		"git hash-object",
		"legacy static archive file set differs",
		"day_log me sql_server translation",
		"python/python3.10安装.html",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("legacy recovery script is missing %q", required)
		}
	}
}

func TestLocalVerifyRejectsStaleGitIndex(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("..", "..", "scripts", "verify.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{
		"git diff --quiet --",
		"git ls-files --others --exclude-standard",
		"git checkout-index --all",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("local verification does not guard the staged tree with %q", required)
		}
	}
}
