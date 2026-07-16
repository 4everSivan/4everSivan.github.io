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
		`{{< hextra/hero-button text="进入文档库" link="/docs/" style="margin-top: 2.2rem; margin-bottom: 2.2rem; display: inline-block;" >}}`,
		"{{< knowledge-stats >}}",
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
		t.Fatalf("home page has %d featured categories, want 6-8", featured)
	}
	if strings.Contains(home, "https://") || strings.Contains(home, "http://") {
		t.Fatal("home page must not add remote assets or links")
	}
	if strings.Contains(home, "首页统计仅包含已通过安全门禁并进入当前构建的内容") {
		t.Fatal("home page must not render the obsolete publication footnote")
	}
	featuredCategories := []struct {
		link string
		dir  string
	}{
		{"/docs/postgresql社区版/", filepath.Join("content", "docs", "Postgresql社区版")},
		{"/docs/编程语言/go/", filepath.Join("content", "docs", "编程语言", "Go")},
		{"/docs/编程语言/python/", filepath.Join("content", "docs", "编程语言", "python")},
		{"/docs/patroni/", filepath.Join("content", "docs", "Patroni")},
		{"/docs/ai/", filepath.Join("content", "docs", "AI")},
		{"/docs/nosql/", filepath.Join("content", "docs", "NoSQL")},
		{"/docs/zabbix/", filepath.Join("content", "docs", "Zabbix")},
	}
	for _, category := range featuredCategories {
		if !strings.Contains(home, `link="`+category.link+`"`) {
			t.Fatalf("home page is missing featured category %s", category.link)
		}
		directory := filepath.Join(root, category.dir)
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
			t.Fatalf("featured category has no published documents: %s", category.dir)
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
		`partial "shortcodes/card"`,
		"data-published-documents",
		"data-published-categories",
		".hextra-home h2 {",
		"margin-top: 4.5rem !important;",
		"margin-bottom: 0.5rem !important;",
		".hextra-home .hextra-cards,",
		".hextra-home .hextra-feature-grid {",
		"margin-top: 0.5rem !important;",
		".hextra-home .hextra-cards + h2,",
		".hextra-home .hextra-feature-grid + h2 {",
		"margin-top: 1rem !important;",
	} {
		if !strings.Contains(stats, required) {
			t.Fatalf("knowledge stats shortcode is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		`a[href*="/docs/"]`,
		`a[href="/docs/"]`,
		".hextra-home em",
	} {
		if strings.Contains(stats, forbidden) {
			t.Fatalf("knowledge stats shortcode contains over-broad or obsolete style %q", forbidden)
		}
	}

	for _, forbidden := range []string{
		filepath.Join(root, "layouts", "hextra-home.html"),
		filepath.Join(root, "assets", "css", "custom.css"),
	} {
		if _, err := os.Stat(forbidden); !os.IsNotExist(err) {
			t.Fatalf("forbidden theme override exists: %s", forbidden)
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
