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
		`title: "SivanHub"`,
		"layout: hextra-home",
		"{{< hextra/hero-headline >}}\nSivanHub\n{{< /hextra/hero-headline >}}",
		"{{< hextra/hero-subtitle >}}",
		`{{< hextra/hero-button text="进入文档库" link="/docs/"`,
		"{{< terminal-panel >}}",
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

	terminalBytes, err := os.ReadFile(filepath.Join(root, "layouts", "_shortcodes", "terminal-panel.html"))
	if err != nil {
		t.Fatal(err)
	}
	terminal := string(terminalBytes)
	for _, required := range []string{
		`site.GetPage "/docs"`,
		"RegularPagesRecursive",
		"terminal-status",
		"STATUS: VERIFIED",
	} {
		if !strings.Contains(terminal, required) {
			t.Fatalf("terminal panel shortcode is missing %q", required)
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
	for _, required := range []string{".hextra-home", "@font-face", "Silkscreen", ".terminal-panel"} {
		if !strings.Contains(customCSS, required) {
			t.Fatalf("custom.css is missing %q", required)
		}
	}
	if strings.Contains(customCSS, "!important") {
		t.Fatal("custom.css must not use !important overrides")
	}

	// 站点固定暗色: 通过 Hextra 官方 custom/head-end.html 钩子清除历史亮色偏好.
	headEndBytes, err := os.ReadFile(filepath.Join(root, "layouts", "_partials", "custom", "head-end.html"))
	if err != nil {
		t.Fatal(err)
	}
	headEnd := string(headEndBytes)
	for _, required := range []string{"color-theme", `"dark"`, "classList"} {
		if !strings.Contains(headEnd, required) {
			t.Fatalf("head-end.html is missing %q", required)
		}
	}

	// 禁用 JS 时 Hextra 的主题脚本不会执行, 需要 baseof.html 覆盖在服务端输出 class="dark".
	baseofBytes, err := os.ReadFile(filepath.Join(root, "layouts", "baseof.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(baseofBytes), `class="dark"`) {
		t.Fatal("baseof.html override must render class=\"dark\" on the html element")
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

	// 像素字体按项目惯例固定版本并校验哈希安装, 产物目录被 Git 忽略.
	versionsBytes, err := os.ReadFile(filepath.Join(root, "config", "versions.env"))
	if err != nil {
		t.Fatal(err)
	}
	versions := string(versionsBytes)
	for _, required := range []string{"SILKSCREEN_VERSION=", "SILKSCREEN_LATIN_400_WOFF2_SHA256=", "SILKSCREEN_LATIN_700_WOFF2_SHA256="} {
		if !strings.Contains(versions, required) {
			t.Fatalf("versions.env is missing %q", required)
		}
	}
	installBytes, err := os.ReadFile(filepath.Join(root, "scripts", "install-frontend-assets.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(installBytes), "@fontsource/silkscreen@${SILKSCREEN_VERSION}") {
		t.Fatal("install-frontend-assets.sh must install the pinned silkscreen font")
	}
	gitignoreBytes, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gitignoreBytes), "/static/fonts/") {
		t.Fatal(".gitignore must exclude installed font artifacts")
	}
}

func TestHugoConfigKeepsNativeArticlePresentation(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("..", "..", "hugo.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Title  string `yaml:"title"`
		Params struct {
			ExternalLinkDecoration bool `yaml:"externalLinkDecoration"`
			Page                   struct {
				Width string `yaml:"width"`
			} `yaml:"page"`
			Theme struct {
				Default       string `yaml:"default"`
				DisplayToggle bool   `yaml:"displayToggle"`
			} `yaml:"theme"`
		} `yaml:"params"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config.Title != "SivanHub" {
		t.Fatalf("site title = %q, want SivanHub", config.Title)
	}
	if !config.Params.ExternalLinkDecoration {
		t.Fatal("Hextra external link decoration is disabled")
	}
	if config.Params.Page.Width != "normal" {
		t.Fatalf("page width = %q, want normal", config.Params.Page.Width)
	}
	if config.Params.Theme.Default != "dark" {
		t.Fatalf("theme default = %q, want dark", config.Params.Theme.Default)
	}
	if config.Params.Theme.DisplayToggle {
		t.Fatal("theme toggle must be hidden for the fixed dark theme")
	}
}
