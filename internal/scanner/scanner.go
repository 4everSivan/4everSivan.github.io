// Package scanner applies deterministic, redacted content rules and a pinned
// Gitleaks process to one discovered Markdown candidate at a time.
package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	pathpkg "path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/4everSivan/4everSivan.github.io/internal/source"
)

type Level string

const (
	LevelBlock   Level = "block"
	LevelWarning Level = "warning"
)

const (
	RulePrivateKey           = "secret.private-key"
	RuleHighConfidenceToken  = "secret.high-confidence-token"
	RuleCredentialAssignment = "secret.credential-assignment"
	RulePrivateNetwork       = "network.private-address"
	RuleAbsoluteLocalPath    = "path.absolute-local"
	RuleFileURL              = "path.file-url"
	RuleRelativePathEscape   = "path.relative-escape"
	RuleLocalResource        = "resource.local"
	RuleRemoteImage          = "resource.remote-image"
	RuleWikiLink             = "syntax.wiki-link"
	RuleDangerousHTML        = "html.dangerous"
	RuleInvalidUTF8          = "content.invalid-utf8"
	RuleBinaryContent        = "content.binary"
	RuleInvalidFrontMatter   = "markdown.invalid-front-matter"
	RuleUnclosedFence        = "markdown.unclosed-fence"
	GitleaksRulePrefix       = "gitleaks."
)

// Finding deliberately omits matched text and sensitive values.
type Finding struct {
	RuleID       string `json:"rule_id" yaml:"rule_id"`
	Level        Level  `json:"level" yaml:"level"`
	RelativePath string `json:"relative_path" yaml:"relative_path"`
	Line         int    `json:"line" yaml:"line"`
	Reason       string `json:"reason" yaml:"reason"`
	Approvable   bool   `json:"approvable" yaml:"approvable"`
}

type Result struct {
	RelativePath string    `json:"relative_path" yaml:"relative_path"`
	SHA256       string    `json:"sha256" yaml:"sha256"`
	Findings     []Finding `json:"findings" yaml:"findings"`
	Completed    bool      `json:"completed" yaml:"completed"`
}

func (r Result) Blocking() []Finding {
	blocking := make([]Finding, 0)
	for _, finding := range r.Findings {
		if finding.Level == LevelBlock {
			blocking = append(blocking, finding)
		}
	}
	return blocking
}

func (r Result) HasBlocking() bool { return len(r.Blocking()) != 0 }

// ExternalScanner is a mandatory independent credential scanner.
type ExternalScanner interface {
	Check(context.Context) error
	ScanData(context.Context, string, []byte) ([]Finding, error)
}

type Engine struct {
	external ExternalScanner
}

func New(external ExternalScanner) *Engine { return &Engine{external: external} }

// Check verifies that the external scanner exists at the required version.
func (e *Engine) Check(ctx context.Context) error {
	if e == nil || e.external == nil {
		return ErrScannerUnavailable
	}
	return e.external.Check(ctx)
}

// Close releases private scanner copies and other temporary resources.
func (e *Engine) Close() error {
	if e == nil || e.external == nil {
		return nil
	}
	closer, ok := e.external.(interface{ Close() error })
	if !ok {
		return nil
	}
	return closer.Close()
}

// Scan reads and revalidates exactly one discovered candidate. An unavailable
// external scanner is a global error, never a per-document exclusion.
func (e *Engine) Scan(ctx context.Context, root string, candidate source.Candidate) (Result, error) {
	data, err := source.Read(root, candidate)
	if err != nil {
		return Result{}, err
	}
	result, err := e.ScanData(ctx, candidate.RelativePath, data)
	if err != nil {
		return Result{}, err
	}
	if err := source.Validate(root, candidate); err != nil {
		return Result{}, err
	}
	return result, nil
}

// ScanData applies both the project rules and Gitleaks to the exact same byte
// slice. The caller cannot supply the identity hash: it is derived here from
// the bytes that both scanners receive.
func (e *Engine) ScanData(ctx context.Context, relativePath string, data []byte) (Result, error) {
	if e == nil || e.external == nil {
		return Result{}, ErrScannerUnavailable
	}
	digest := sha256.Sum256(data)
	result := ScanBytes(relativePath, hex.EncodeToString(digest[:]), data)
	externalFindings, err := e.external.ScanData(ctx, relativePath, data)
	if err != nil {
		return Result{}, err
	}
	result.Findings = append(result.Findings, externalFindings...)
	result.Findings = normalizeFindings(result.Findings)
	result.Completed = true
	return result, nil
}

// ScanBytes applies project rules to synthetic or already controlled bytes.
// Production source scanning should use Engine.Scan so Gitleaks and source
// change detection cannot be skipped.
func ScanBytes(relativePath, sha256 string, data []byte) Result {
	result := Result{RelativePath: relativePath, SHA256: sha256}
	add := func(rule string, level Level, line int, reason string, approvable bool) {
		result.Findings = append(result.Findings, Finding{
			RuleID:       rule,
			Level:        level,
			RelativePath: relativePath,
			Line:         line,
			Reason:       reason,
			Approvable:   approvable,
		})
	}

	if !utf8.Valid(data) {
		add(RuleInvalidUTF8, LevelBlock, 0, "内容不是有效 UTF-8 编码", false)
		result.Findings = normalizeFindings(result.Findings)
		return result
	}
	if index := strings.IndexByte(string(data), 0); index >= 0 {
		add(RuleBinaryContent, LevelBlock, lineAt(data, index), "内容包含二进制空字节", false)
	}

	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if startsFrontMatter(lines) && !hasClosedFrontMatter(lines) {
		add(RuleInvalidFrontMatter, LevelBlock, 1, "Hugo front matter 未闭合", false)
	}
	if line := unclosedFenceLine(lines); line != 0 {
		add(RuleUnclosedFence, LevelBlock, line, "Markdown 代码围栏未闭合", false)
	}

	for i, line := range lines {
		lineNumber := i + 1
		if privateKeyPattern.MatchString(line) {
			add(RulePrivateKey, LevelBlock, lineNumber, "检测到私钥边界", false)
		}
		if highConfidenceTokenPattern.MatchString(line) {
			add(RuleHighConfidenceToken, LevelBlock, lineNumber, "检测到高置信度凭据格式", false)
		}
		if credentialAssignmentPattern.MatchString(line) || credentialURLPattern.MatchString(line) {
			add(RuleCredentialAssignment, LevelBlock, lineNumber, "检测到密码或 token 配置赋值", false)
		}
		if privateNetworkPattern.MatchString(line) || privateIPv6Pattern.MatchString(line) || privateHostPattern.MatchString(line) {
			add(RulePrivateNetwork, LevelBlock, lineNumber, "检测到内网或本机地址", true)
		}
		if absoluteLocalPathPattern.MatchString(line) || windowsLocalPathPattern.MatchString(line) {
			add(RuleAbsoluteLocalPath, LevelBlock, lineNumber, "检测到绝对本地路径", false)
		}
		if fileURLPattern.MatchString(line) {
			add(RuleFileURL, LevelBlock, lineNumber, "检测到 file URL", false)
		}
		if wikiLinkPattern.MatchString(line) {
			add(RuleWikiLink, LevelBlock, lineNumber, "检测到无法确定转换语义的 Wiki 链接", false)
		}
		if dangerousHTMLPattern.MatchString(line) || dangerousHTMLAttributePattern.MatchString(line) {
			add(RuleDangerousHTML, LevelBlock, lineNumber, "检测到危险 HTML", false)
		}

		for _, match := range markdownLinkPattern.FindAllStringSubmatch(line, -1) {
			if len(match) < 3 {
				continue
			}
			isImage := match[1] == "!"
			scanReference(relativePath, strings.TrimSpace(match[2]), isImage, lineNumber, add)
		}
		if match := markdownReferenceDefinitionPattern.FindStringSubmatch(line); len(match) >= 2 {
			scanReference(relativePath, strings.TrimSpace(match[1]), false, lineNumber, add)
		}
		for _, match := range htmlResourcePattern.FindAllStringSubmatch(line, -1) {
			if len(match) < 5 {
				continue
			}
			isImage := strings.EqualFold(match[1], "src")
			resource := ""
			for _, candidate := range match[2:] {
				if candidate != "" {
					resource = candidate
					break
				}
			}
			scanReference(relativePath, strings.TrimSpace(resource), isImage, lineNumber, add)
		}
	}

	result.Findings = normalizeFindings(result.Findings)
	return result
}

var (
	privateKeyPattern                  = regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`)
	highConfidenceTokenPattern         = regexp.MustCompile(`(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|AKIA[0-9A-Z]{16}|xox[baprs]-[A-Za-z0-9-]{10,}|sk-(proj-)?[A-Za-z0-9_-]{20,})`)
	credentialAssignmentPattern        = regexp.MustCompile(`(?i)\b(password|passwd|pwd|token|secret|api[_-]?key)\b\s*[:=]\s*(?:"[^"\r\n]{4,}"|'[^'\r\n]{4,}'|[^[:space:]"']{4,})`)
	credentialURLPattern               = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://[^/[:space:]:@]+:[^/[:space:]@]{4,}@`)
	privateNetworkPattern              = regexp.MustCompile(`(?:^|[^0-9])(?:10\.(?:[0-9]{1,3}\.){2}[0-9]{1,3}|172\.(?:1[6-9]|2[0-9]|3[01])\.(?:[0-9]{1,3}\.)[0-9]{1,3}|192\.168\.(?:[0-9]{1,3}\.)[0-9]{1,3}|127\.(?:[0-9]{1,3}\.){2}[0-9]{1,3})(?:[^0-9]|$)`)
	privateIPv6Pattern                 = regexp.MustCompile(`(?i)(?:^|[^A-Fa-f0-9:])(?:::1|f[cd][A-Fa-f0-9]{0,2}:|fe[89ab][A-Fa-f0-9]?:)[A-Fa-f0-9:]*`)
	privateHostPattern                 = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9.-])(?:localhost|[A-Za-z0-9-]+\.(?:local|internal|lan))(?:[^A-Za-z0-9.-]|$)`)
	absoluteLocalPathPattern           = regexp.MustCompile(`(?:^|[[:space:]\("'=:])/(?:Users|home|private|Volumes|tmp|var|etc|opt|usr|Library|Applications|srv|data|mnt|root|run|boot|dev|proc|sys|workspace)/[^[:space:]\)"'>]+`)
	windowsLocalPathPattern            = regexp.MustCompile(`(?i)(?:^|[[:space:]\("'=:])[A-Z]:[\\/][^[:space:]\)"']+`)
	fileURLPattern                     = regexp.MustCompile(`(?i)\bfile://`)
	wikiLinkPattern                    = regexp.MustCompile(`\[\[[^\]]+\]\]`)
	dangerousHTMLPattern               = regexp.MustCompile(`(?i)<\s*/?\s*(script|iframe|object|embed|form|base)(?:\s|>|/)`)
	dangerousHTMLAttributePattern      = regexp.MustCompile(`(?i)(?:\son[a-z]+\s*=|javascript\s*:)`)
	markdownLinkPattern                = regexp.MustCompile(`(!?)\[[^\]]*\]\(\s*<?([^\s)>]+)>?(?:\s+[^)]*)?\)`)
	markdownReferenceDefinitionPattern = regexp.MustCompile(`^\s*\[[^\]]+\]:\s*<?([^\s>]+)>?`)
	htmlResourcePattern                = regexp.MustCompile(`(?i)\b(src|href)\s*=\s*(?:"([^"]+)"|'([^']+)'|([^[:space:]>]+))`)
)

func scanReference(relativePath, raw string, image bool, line int, add func(string, Level, int, string, bool)) {
	if raw == "" || strings.HasPrefix(raw, "#") {
		return
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		add(RuleLocalResource, LevelBlock, line, "本地资源引用无法解析", false)
		return
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		if image {
			add(RuleRemoteImage, LevelWarning, line, "远程图片可发布但可用性需单独验证", false)
		}
		return
	case "mailto", "tel":
		if !image {
			return
		}
	case "file":
		add(RuleFileURL, LevelBlock, line, "检测到 file URL", false)
		return
	case "":
		// Continue with local path validation.
	default:
		add(RuleLocalResource, LevelBlock, line, "检测到不受支持的资源协议", false)
		return
	}

	unescaped, err := url.PathUnescape(parsed.Path)
	if err != nil || unescaped == "" {
		if err != nil {
			add(RuleLocalResource, LevelBlock, line, "本地资源路径无法解析", false)
		}
		return
	}
	if strings.HasPrefix(unescaped, "/") || strings.HasPrefix(unescaped, "~") {
		add(RuleAbsoluteLocalPath, LevelBlock, line, "检测到绝对本地路径", false)
		return
	}
	joined := pathpkg.Clean(pathpkg.Join(pathpkg.Dir(relativePath), strings.ReplaceAll(unescaped, "\\", "/")))
	if joined == ".." || strings.HasPrefix(joined, "../") {
		add(RuleRelativePathEscape, LevelBlock, line, "相对资源路径越过源目录边界", false)
		return
	}
	if image {
		add(RuleLocalResource, LevelBlock, line, "本地图片不在 Markdown 采集范围内", false)
		return
	}
	ext := strings.ToLower(pathpkg.Ext(unescaped))
	if ext != "" && ext != ".md" {
		add(RuleLocalResource, LevelBlock, line, "本地非 Markdown 资源不在采集范围内", false)
	}
}

func normalizeFindings(findings []Finding) []Finding {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].RelativePath != findings[j].RelativePath {
			return findings[i].RelativePath < findings[j].RelativePath
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].RuleID < findings[j].RuleID
	})
	unique := findings[:0]
	for _, finding := range findings {
		if len(unique) > 0 {
			last := unique[len(unique)-1]
			if last.RelativePath == finding.RelativePath && last.Line == finding.Line && last.RuleID == finding.RuleID {
				continue
			}
		}
		unique = append(unique, finding)
	}
	return unique
}

func startsFrontMatter(lines []string) bool {
	return len(lines) > 0 && strings.TrimSpace(strings.TrimPrefix(lines[0], "\ufeff")) == "---"
}

func hasClosedFrontMatter(lines []string) bool {
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" || trimmed == "..." {
			return true
		}
	}
	return false
}

func unclosedFenceLine(lines []string) int {
	marker := ""
	openedAt := 0
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		candidate := ""
		if strings.HasPrefix(trimmed, "```") {
			candidate = "```"
		} else if strings.HasPrefix(trimmed, "~~~") {
			candidate = "~~~"
		}
		if candidate == "" {
			continue
		}
		if marker == "" {
			marker = candidate
			openedAt = index + 1
		} else if marker == candidate {
			marker = ""
			openedAt = 0
		}
	}
	return openedAt
}

func lineAt(data []byte, index int) int {
	if index < 0 {
		return 0
	}
	return 1 + strings.Count(string(data[:index]), "\n")
}

var ErrScannerUnavailable = errors.New("required external scanner is unavailable")

func scannerError(operation string, err error) error {
	if err == nil {
		return nil
	}
	// The adapter never attaches process stdout/stderr to err, so preserving the
	// typed operational error here cannot reveal a detected value.
	return fmt.Errorf("%s: %w", operation, errors.Join(ErrScannerUnavailable, err))
}
