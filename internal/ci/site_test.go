package ci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestHomeEditorialKnowledgeNavigation(t *testing.T) {
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
		"{{< home-hero >}}",
		"{{< knowledge-stats >}}",
		"{{< category-wall >}}",
		"{{< recent-docs ",
		"{{< hextra/feature-grid ",
		"{{< hextra/feature-card ",
	} {
		if !strings.Contains(home, required) {
			t.Fatalf("home page is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"terminal-panel",
		"hero-badge",
		"hero-headline",
		"hero-subtitle",
		"hero-button",
	} {
		if strings.Contains(home, forbidden) {
			t.Fatalf("home page must not use the retired HUD element %q", forbidden)
		}
	}

	// Hero 使用独立短代码: Goldmark 默认不渲染 Markdown 内联原始 HTML.
	heroBytes, err := os.ReadFile(filepath.Join(root, "layouts", "_shortcodes", "home-hero.html"))
	if err != nil {
		t.Fatal(err)
	}
	hero := string(heroBytes)
	for _, required := range []string{
		`class="home-hero`,
		`<h1 class="home-title">SivanHub</h1>`,
		`class="home-cta"`,
		`"/docs/" | relURL`,
	} {
		if !strings.Contains(hero, required) {
			t.Fatalf("home hero shortcode is missing %q", required)
		}
	}
	featured := strings.Count(home, "{{< hextra/feature-card ")
	if featured != 3 {
		t.Fatalf("home page has %d featured cards, want 3", featured)
	}
	if strings.Contains(home, "https://") || strings.Contains(home, "http://") {
		t.Fatal("home page must not add remote assets or links")
	}
	if strings.Contains(home, "首页统计仅包含已通过安全门禁并进入当前构建的内容") {
		t.Fatal("home page must not render the obsolete publication footnote")
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

	// HUD 终端面板已随编辑风改版退役, 短代码文件必须一并移除.
	if _, err := os.Stat(filepath.Join(root, "layouts", "_shortcodes", "terminal-panel.html")); !os.IsNotExist(err) {
		t.Fatal("terminal panel shortcode must be removed")
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
	for _, required := range []string{".hextra-home", "@font-face", "Source Serif 4", ".category-wall"} {
		if !strings.Contains(customCSS, required) {
			t.Fatalf("custom.css is missing %q", required)
		}
	}
	for _, forbidden := range []string{"!important", "Silkscreen", "terminal-panel"} {
		if strings.Contains(customCSS, forbidden) {
			t.Fatalf("custom.css must not contain %q", forbidden)
		}
	}

	// 站点主题跟随系统: 通过 Hextra 官方 custom/head-end.html 钩子
	// 把历史强制 dark 偏好统一迁移为 system.
	headEndBytes, err := os.ReadFile(filepath.Join(root, "layouts", "_partials", "custom", "head-end.html"))
	if err != nil {
		t.Fatal(err)
	}
	headEnd := string(headEndBytes)
	for _, required := range []string{"color-theme", `"system"`} {
		if !strings.Contains(headEnd, required) {
			t.Fatalf("head-end.html is missing %q", required)
		}
	}

	// 主题由客户端脚本按 localStorage/系统偏好设置, baseof.html 不得在服务端写死主题 class.
	baseofBytes, err := os.ReadFile(filepath.Join(root, "layouts", "baseof.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(baseofBytes), `class="dark"`) {
		t.Fatal("baseof.html override must not hardcode class=\"dark\" on the html element")
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

	// 衬线字体按项目惯例固定版本并校验哈希安装, 产物目录被 Git 忽略.
	versionsBytes, err := os.ReadFile(filepath.Join(root, "config", "versions.env"))
	if err != nil {
		t.Fatal(err)
	}
	versions := string(versionsBytes)
	for _, required := range []string{
		"SOURCE_SERIF_4_VERSION=",
		"SOURCE_SERIF_4_LATIN_400_WOFF2_SHA256=",
		"SOURCE_SERIF_4_LATIN_500_WOFF2_SHA256=",
		"SOURCE_SERIF_4_LATIN_600_WOFF2_SHA256=",
	} {
		if !strings.Contains(versions, required) {
			t.Fatalf("versions.env is missing %q", required)
		}
	}
	installBytes, err := os.ReadFile(filepath.Join(root, "scripts", "install-frontend-assets.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(installBytes), "@fontsource/source-serif-4@${SOURCE_SERIF_4_VERSION}") {
		t.Fatal("install-frontend-assets.sh must install the pinned Source Serif 4 font")
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
	if config.Params.Theme.Default != "system" {
		t.Fatalf("theme default = %q, want system", config.Params.Theme.Default)
	}
	if config.Params.Theme.DisplayToggle {
		t.Fatal("theme toggle must be hidden; theme follows the system preference")
	}
}
