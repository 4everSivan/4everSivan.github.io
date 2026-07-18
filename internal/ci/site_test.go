package ci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestHomeUsesNativeHextraKnowledgeNavigation(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..")
	homeBytes, err := os.ReadFile(filepath.Join(root, "content", "_index.md"))
	if err != nil {
		t.Fatal(err)
	}
	home := string(homeBytes)
	for _, required := range []string{
		"layout: hextra-home",
		"{{< hextra/hero-headline >}}",
		"{{< hextra/hero-subtitle >}}",
		`{{< hextra/hero-button text="进入文档库" link="/docs/"`,
		"{{< knowledge-stats >}}",
		`{{< category-wall pinned="Postgresql社区版,AI,Patroni,编程语言" >}}`,
		"{{< recent-docs ",
		"{{< hextra/feature-grid ",
		"{{< hextra/feature-card ",
		`{{< cards cols="2" >}}`,
		"查看全部文档",
		"全文搜索",
	} {
		if !strings.Contains(home, required) {
			t.Fatalf("home page is missing %q", required)
		}
	}
	featured := strings.Count(home, "{{< hextra/feature-card ")
	if featured < 6 || featured > 8 {
		t.Fatalf("home page has %d featured cards, want 6-8", featured)
	}
	if strings.Contains(home, "https://") || strings.Contains(home, "http://") {
		t.Fatal("home page must not add remote assets or links")
	}
	if strings.Contains(home, "首页统计仅包含已通过安全门禁并进入当前构建的内容") {
		t.Fatal("home page must not render the obsolete publication footnote")
	}
	pinnedCategories := []string{"Postgresql社区版", "AI", "Patroni", "编程语言"}
	for _, category := range pinnedCategories {
		directory := filepath.Join(root, "content", "docs", category)
		hasDocument := false
		err := filepath.WalkDir(directory, func(filename string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() && filepath.Ext(filename) == ".md" && entry.Name() != "_index.md" {
				hasDocument = true
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if !hasDocument {
			t.Fatalf("pinned category has no published documents: %s", category)
		}
	}

	statsBytes, err := os.ReadFile(filepath.Join(root, "layouts", "_shortcodes", "knowledge-stats.html"))
	if err != nil {
		t.Fatal(err)
	}
	stats := string(statsBytes)
	for _, required := range []string{
		`site.GetPage "/docs"`,
		"RegularPagesRecursive",
		`eq .Kind "section"`,
		"errorf",
		"data-knowledge-stats",
		"data-published-documents",
		"data-published-categories",
		"data-count-to",
	} {
		if !strings.Contains(stats, required) {
			t.Fatalf("knowledge stats shortcode is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"<style",
		"!important",
		`a[href*="/docs/"]`,
		`a[href="/docs/"]`,
		".hextra-home em",
	} {
		if strings.Contains(stats, forbidden) {
			t.Fatalf("knowledge stats shortcode contains inline style or obsolete selector %q", forbidden)
		}
	}

	// 首页样式集中在 assets/css/custom.css 并以 .hextra-home 作用域隔离;
	// hextra-home 布局覆盖仅负责提供该作用域类.
	homeLayoutBytes, err := os.ReadFile(filepath.Join(root, "layouts", "hextra-home.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(homeLayoutBytes), "hextra-home") {
		t.Fatal("home layout must provide the hextra-home scoping class")
	}
	customCSSBytes, err := os.ReadFile(filepath.Join(root, "assets", "css", "custom.css"))
	if err != nil {
		t.Fatal(err)
	}
	customCSS := string(customCSSBytes)
	if !strings.Contains(customCSS, ".hextra-home") {
		t.Fatal("custom.css must scope home page styles under .hextra-home")
	}
	if strings.Contains(customCSS, "!important") {
		t.Fatal("custom.css must not use !important overrides")
	}

	wallBytes, err := os.ReadFile(filepath.Join(root, "layouts", "_shortcodes", "category-wall.html"))
	if err != nil {
		t.Fatal(err)
	}
	wall := string(wallBytes)
	for _, required := range []string{
		`site.GetPage "/docs"`,
		".Sections",
		"RegularPagesRecursive",
		`"Lastmod"`,
		`.Get "pinned"`,
	} {
		if !strings.Contains(wall, required) {
			t.Fatalf("category wall shortcode is missing %q", required)
		}
	}

	recentBytes, err := os.ReadFile(filepath.Join(root, "layouts", "_shortcodes", "recent-docs.html"))
	if err != nil {
		t.Fatal(err)
	}
	recent := string(recentBytes)
	for _, required := range []string{
		"RegularPagesRecursive",
		`"Lastmod"`,
		`.Get "limit"`,
	} {
		if !strings.Contains(recent, required) {
			t.Fatalf("recent docs shortcode is missing %q", required)
		}
	}

	revealBytes, err := os.ReadFile(filepath.Join(root, "assets", "js", "head", "reveal.js"))
	if err != nil {
		t.Fatal(err)
	}
	reveal := string(revealBytes)
	for _, required := range []string{
		"classList.add",
		"IntersectionObserver",
		"prefers-reduced-motion",
		"data-count-to",
	} {
		if !strings.Contains(reveal, required) {
			t.Fatalf("reveal.js is missing %q", required)
		}
	}
}

func TestHugoConfigKeepsNativeArticlePresentation(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("..", "..", "hugo.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Params struct {
			ExternalLinkDecoration bool `yaml:"externalLinkDecoration"`
			Page                   struct {
				Width string `yaml:"width"`
			} `yaml:"page"`
		} `yaml:"params"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if !config.Params.ExternalLinkDecoration {
		t.Fatal("Hextra external link decoration is disabled")
	}
	if config.Params.Page.Width != "normal" {
		t.Fatalf("page width = %q, want normal", config.Params.Page.Width)
	}
}
