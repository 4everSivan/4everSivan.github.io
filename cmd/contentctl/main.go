package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/4everSivan/4everSivan.github.io/internal/approval"
	"github.com/4everSivan/4everSivan.github.io/internal/report"
	"github.com/4everSivan/4everSivan.github.io/internal/scanner"
	"github.com/4everSivan/4everSivan.github.io/internal/snapshot"
	"github.com/4everSivan/4everSivan.github.io/internal/source"
	"github.com/4everSivan/4everSivan.github.io/internal/transform"
)

type commonOptions struct {
	projectRoot    string
	sourceRoot     string
	gitleaksBinary string
	gitleaksSHA256 string
}

type runtimeConfig struct {
	sourceRoot     string
	gitleaksSHA256 string
	enforceSource  bool
}

type scanRun struct {
	manifest      source.Manifest
	results       []scanner.Result
	allowlist     approval.Allowlist
	scannedAt     time.Time
	rulesetSHA256 string
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "contentctl 执行失败: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	gitleaksSHA256, err := scanner.RequiredGitleaksSHA256()
	if err != nil {
		return err
	}
	return runConfigured(ctx, args, stdout, runtimeConfig{
		sourceRoot: source.DefaultRoot, gitleaksSHA256: gitleaksSHA256, enforceSource: true,
	})
}

func runConfigured(ctx context.Context, args []string, stdout io.Writer, runtime runtimeConfig) error {
	if len(args) == 0 {
		return errors.New("需要子命令: scan, sync, approve 或 verify")
	}
	switch args[0] {
	case "scan":
		return runScan(ctx, args[1:], stdout, runtime)
	case "sync":
		return runSync(ctx, args[1:], stdout, runtime)
	case "approve":
		return runApprove(ctx, args[1:], stdout, runtime)
	case "verify":
		return runVerify(ctx, args[1:], stdout, runtime)
	case "version":
		fmt.Fprintf(stdout, "contentctl (gitleaks %s)\n", scanner.RequiredGitleaksVersion)
		return nil
	default:
		return fmt.Errorf("未知子命令 %q", args[0])
	}
}

func runScan(ctx context.Context, args []string, stdout io.Writer, runtime runtimeConfig) error {
	options, err := parseCommon("scan", args, true, runtime)
	if err != nil {
		return err
	}
	scan, err := scanSource(ctx, options)
	if err != nil {
		return err
	}
	if err := writeLocalReports(options.projectRoot, scan); err != nil {
		return err
	}
	passed, excluded, warnings := counts(scan.results, scan.allowlist, scan.scannedAt)
	fmt.Fprintf(stdout, "扫描完成: 候选=%d, 通过=%d, 排除=%d, 警告=%d\n", len(scan.results), passed, excluded, warnings)
	return nil
}

func runSync(ctx context.Context, args []string, stdout io.Writer, runtime runtimeConfig) error {
	options, err := parseCommon("sync", args, true, runtime)
	if err != nil {
		return err
	}
	scan, err := scanSource(ctx, options)
	if err != nil {
		return err
	}

	files, contentManifest, err := buildSnapshot(ctx, options, &scan)
	if err != nil {
		return err
	}
	if err := writeLocalReports(options.projectRoot, scan); err != nil {
		return err
	}

	destination := filepath.Join(options.projectRoot, "content", "docs")
	validateSource := func() error {
		if err := validateManifestUnchanged(scan.manifest); err != nil {
			return err
		}
		return validateRulesetUnchanged(options.projectRoot, scan.rulesetSHA256)
	}
	if err := snapshot.ReplaceValidated(destination, files, validateSource); err != nil {
		return err
	}
	passed, excluded, warnings := counts(scan.results, scan.allowlist, scan.scannedAt)
	fmt.Fprintf(stdout, "同步完成: 候选=%d, 已同步=%d, 已排除=%d, 警告=%d, 清单文档=%d\n", len(scan.results), passed, excluded, warnings, len(contentManifest.Documents))
	return nil
}

func runApprove(ctx context.Context, args []string, stdout io.Writer, runtime runtimeConfig) error {
	set := flag.NewFlagSet("approve", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	options, err := addCommonFlags(set)
	if err != nil {
		return err
	}
	relativePath := set.String("path", "", "排除报告中的文档相对路径")
	fingerprint := set.String("finding", "", "排除报告中的 finding fingerprint")
	reason := set.String("reason", "", "不含敏感值的复核理由")
	expires := set.String("expires", "", "RFC3339 格式的批准失效时间")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 || *relativePath == "" || *fingerprint == "" || *reason == "" || *expires == "" {
		return errors.New("approve 必须提供 --path、--finding、--reason 和 --expires")
	}
	finalOptions, err := finalizeOptions(*options, true, runtime)
	if err != nil {
		return err
	}
	options = &finalOptions
	expiresAt, err := time.Parse(time.RFC3339, *expires)
	if err != nil {
		return errors.New("--expires 必须是有效 RFC3339 时间")
	}
	approvedAt := time.Now().UTC()
	if !expiresAt.After(approvedAt) {
		return errors.New("批准失效时间必须晚于当前时间")
	}

	scan, err := scanSource(ctx, *options)
	if err != nil {
		return err
	}
	if err := writeLocalReports(options.projectRoot, scan); err != nil {
		return err
	}
	candidateIndex := -1
	for index := range scan.manifest.Candidates {
		if scan.manifest.Candidates[index].RelativePath == *relativePath {
			candidateIndex = index
			break
		}
	}
	if candidateIndex < 0 {
		return errors.New("指定文档不在当前 Markdown 候选清单中")
	}
	candidate := scan.manifest.Candidates[candidateIndex]
	result := scan.results[candidateIndex]
	var selected *scanner.Finding
	for index := range result.Findings {
		if approval.FindingFingerprint(result.Findings[index]) == *fingerprint {
			selected = &result.Findings[index]
			break
		}
	}
	if selected == nil {
		return errors.New("当前完整扫描中不存在指定 finding；内容或规则可能已变化")
	}
	data, err := source.Read(scan.manifest.Root, candidate)
	if err != nil {
		return err
	}
	converted, err := transform.Document(candidate.RelativePath, data)
	if err != nil {
		return errors.New("指定文档无法安全转换, 不可批准")
	}
	engine := newScanner(*options)
	defer engine.Close()
	outputResult, err := engine.ScanData(ctx, candidate.RelativePath, converted)
	if err != nil {
		return err
	}
	outputFinding, err := pairOutputFinding(result, *selected, outputResult)
	if err != nil {
		return err
	}
	entry, err := scan.allowlist.Add(approval.Request{
		SourceResult: result, SourceFinding: *selected,
		OutputResult: outputResult, OutputFinding: outputFinding,
		Reason: *reason, ApprovedAt: approvedAt, ExpiresAt: expiresAt,
	})
	if err != nil {
		return err
	}
	if err := validateManifestUnchanged(scan.manifest); err != nil {
		return err
	}
	if err := validateRulesetUnchanged(options.projectRoot, scan.rulesetSHA256); err != nil {
		return err
	}
	if err := scan.allowlist.Save(filepath.Join(options.projectRoot, approval.ConfigPath)); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "批准已记录: path=%s, rule=%s, expires=%s；重新运行完整 sync 后才会进入快照\n", entry.Path, entry.RuleID, entry.ExpiresAt.Format(time.RFC3339))
	return nil
}

func runVerify(ctx context.Context, args []string, stdout io.Writer, runtime runtimeConfig) error {
	options, err := parseCommon("verify", args, false, runtime)
	if err != nil {
		return err
	}
	contentRoot := filepath.Join(options.projectRoot, "content", "docs")
	contentManifest, err := snapshot.LoadManifest(filepath.Join(contentRoot, snapshot.ManifestPath))
	if err != nil {
		return err
	}
	if contentManifest.GitleaksVersion != scanner.RequiredGitleaksVersion {
		return errors.New("内容清单的 Gitleaks 版本与项目固定版本不一致")
	}
	rulesetHash, err := rulesetFingerprint(options.projectRoot)
	if err != nil {
		return err
	}
	if contentManifest.RulesetSHA256 != rulesetHash {
		return errors.New("扫描规则或批准配置已变化，需要重新执行 sync")
	}
	if err := snapshot.VerifyManifest(contentRoot, contentManifest); err != nil {
		return err
	}
	if err := verifyContentBoundary(filepath.Join(options.projectRoot, "content"), contentManifest); err != nil {
		return err
	}
	allowlist, err := approval.Load(filepath.Join(options.projectRoot, approval.ConfigPath))
	if err != nil {
		return err
	}
	discovered, err := source.Discover(filepath.Join(options.projectRoot, "content"))
	if err != nil {
		return err
	}
	engine := newScanner(options)
	defer engine.Close()
	if err := engine.Check(ctx); err != nil {
		return err
	}
	documents := make(map[string]snapshot.ManifestDocument, len(contentManifest.Documents))
	for _, document := range contentManifest.Documents {
		documents[document.Path] = document
	}
	indexes := make(map[string]struct{}, len(contentManifest.GeneratedIndexes))
	for _, index := range contentManifest.GeneratedIndexes {
		indexes[index.Path] = struct{}{}
	}
	at := time.Now().UTC()
	scaffoldCount := 0
	verifiedDocuments := 0
	verifiedIndexes := 0
	for _, candidate := range discovered.Candidates {
		result, err := engine.Scan(ctx, discovered.Root, candidate)
		if err != nil {
			return err
		}
		if candidate.RelativePath == "_index.md" {
			if result.HasBlocking() {
				return errors.New("站点入口未通过安全复检")
			}
			scaffoldCount++
			continue
		}
		if !strings.HasPrefix(candidate.RelativePath, "docs/") {
			return fmt.Errorf("发现未受控的站点 Markdown: %s", candidate.RelativePath)
		}
		relativePath := strings.TrimPrefix(candidate.RelativePath, "docs/")
		if path.Base(relativePath) == "_index.md" {
			if _, ok := indexes[relativePath]; !ok {
				return fmt.Errorf("分类索引未在内容清单中登记: %s", relativePath)
			}
			if result.HasBlocking() {
				return fmt.Errorf("生成的分类索引未通过安全复检: %s", relativePath)
			}
			verifiedIndexes++
			continue
		}
		document, ok := documents[relativePath]
		if !ok {
			return fmt.Errorf("快照文档未在内容清单中登记: %s", relativePath)
		}
		result.RelativePath = relativePath
		for index := range result.Findings {
			result.Findings[index].RelativePath = relativePath
		}
		if blocking := allowlist.UnapprovedOutputBlocking(result, at); len(blocking) != 0 {
			return fmt.Errorf("受控文档安全复检失败: path=%s, rule=%s, line=%d", relativePath, blocking[0].RuleID, blocking[0].Line)
		}
		if !equalStrings(document.ApprovedRules, approvedOutputRules(result, allowlist, at)) {
			return fmt.Errorf("内容清单批准规则与实际输出批准不一致: %s", relativePath)
		}
		verifiedDocuments++
	}
	if scaffoldCount != 1 || verifiedDocuments != len(contentManifest.Documents) || verifiedIndexes != len(contentManifest.GeneratedIndexes) {
		return errors.New("站点入口、受控文档或分类索引复检数量与内容清单不一致")
	}
	if err := validateRulesetUnchanged(options.projectRoot, rulesetHash); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "验证完成: 站点入口=%d, 受控文档=%d, 分类索引=%d, 快照与逐文件安全复检全部通过\n", scaffoldCount, verifiedDocuments, verifiedIndexes)
	return nil
}

func parseCommon(name string, args []string, withSource bool, runtime runtimeConfig) (commonOptions, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	options, err := addCommonFlags(set)
	if err != nil {
		return commonOptions{}, err
	}
	if err := set.Parse(args); err != nil {
		return commonOptions{}, err
	}
	if set.NArg() != 0 {
		return commonOptions{}, errors.New("存在无法识别的位置参数")
	}
	return finalizeOptions(*options, withSource, runtime)
}

func addCommonFlags(set *flag.FlagSet) (*commonOptions, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	options := &commonOptions{}
	set.StringVar(&options.projectRoot, "project-root", workingDirectory, "项目根目录")
	return options, nil
}

func finalizeOptions(options commonOptions, withSource bool, runtime runtimeConfig) (commonOptions, error) {
	projectRoot, err := filepath.Abs(options.projectRoot)
	if err != nil {
		return commonOptions{}, err
	}
	projectRoot, err = filepath.EvalSymlinks(projectRoot)
	if err != nil {
		return commonOptions{}, fmt.Errorf("解析项目根目录: %w", err)
	}
	options.projectRoot = projectRoot
	options.gitleaksBinary = filepath.Join(projectRoot, ".local", "bin", "gitleaks")
	options.gitleaksSHA256 = runtime.gitleaksSHA256
	if withSource {
		if runtime.sourceRoot == "" {
			return commonOptions{}, errors.New("固定源目录未配置")
		}
		if runtime.enforceSource && filepath.Clean(runtime.sourceRoot) != filepath.Clean(source.DefaultRoot) {
			return commonOptions{}, errors.New("源目录必须固定为 /Users/sivan/work/学习文档")
		}
		options.sourceRoot, err = filepath.Abs(runtime.sourceRoot)
		if err != nil {
			return commonOptions{}, err
		}
		options.sourceRoot, err = filepath.EvalSymlinks(options.sourceRoot)
		if err != nil {
			return commonOptions{}, errors.New("无法解析固定源目录")
		}
		if pathsOverlap(options.projectRoot, options.sourceRoot) {
			return commonOptions{}, errors.New("项目目录与只读源目录不得重叠")
		}
	}
	return options, nil
}

func pathsOverlap(left, right string) bool {
	contains := func(root, candidate string) bool {
		relative, err := filepath.Rel(root, candidate)
		if err != nil {
			return false
		}
		return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
	}
	return contains(left, right) || contains(right, left)
}

func newScanner(options commonOptions) *scanner.Engine {
	runner := scanner.NewGitleaksRunner(options.gitleaksBinary, filepath.Join(options.projectRoot, ".gitleaks.toml"), options.gitleaksSHA256)
	return scanner.New(runner)
}

func scanSource(ctx context.Context, options commonOptions) (scanRun, error) {
	rulesetHash, err := rulesetFingerprint(options.projectRoot)
	if err != nil {
		return scanRun{}, err
	}
	allowlist, err := approval.Load(filepath.Join(options.projectRoot, approval.ConfigPath))
	if err != nil {
		return scanRun{}, err
	}
	manifest, err := source.Discover(options.sourceRoot)
	if err != nil {
		return scanRun{}, err
	}
	engine := newScanner(options)
	defer engine.Close()
	if err := engine.Check(ctx); err != nil {
		return scanRun{}, err
	}
	results := make([]scanner.Result, 0, len(manifest.Candidates))
	for _, candidate := range manifest.Candidates {
		result, err := engine.Scan(ctx, manifest.Root, candidate)
		if err != nil {
			return scanRun{}, err
		}
		results = append(results, result)
	}
	if len(results) != len(manifest.Candidates) {
		return scanRun{}, errors.New("候选文档未全部产生扫描结果")
	}
	if err := validateRulesetUnchanged(options.projectRoot, rulesetHash); err != nil {
		return scanRun{}, err
	}
	return scanRun{
		manifest: manifest, results: results, allowlist: allowlist,
		scannedAt: time.Now().UTC(), rulesetSHA256: rulesetHash,
	}, nil
}

func writeLocalReports(projectRoot string, scan scanRun) error {
	exclusions, err := report.FromResults(scan.results, scan.scannedAt, scan.allowlist)
	if err != nil {
		return err
	}
	inventory, err := report.InventoryFromResults(scan.results, scan.scannedAt, scan.allowlist)
	if err != nil {
		return err
	}
	if err := exclusions.Save(filepath.Join(projectRoot, report.LocalPath)); err != nil {
		return err
	}
	if err := inventory.Save(filepath.Join(projectRoot, report.InventoryPath)); err != nil {
		return err
	}
	return nil
}

func buildSnapshot(ctx context.Context, options commonOptions, scan *scanRun) ([]snapshot.File, snapshot.Manifest, error) {
	engine := newScanner(options)
	defer engine.Close()
	if err := engine.Check(ctx); err != nil {
		return nil, snapshot.Manifest{}, err
	}
	documents := make(map[string][]byte)
	manifestDocuments := make([]snapshot.ManifestDocument, 0)
	for index, result := range scan.results {
		if len(scan.allowlist.UnapprovedBlocking(result, scan.scannedAt)) != 0 {
			continue
		}
		candidate := scan.manifest.Candidates[index]
		if path.Base(candidate.RelativePath) == "_index.md" {
			scan.results[index] = addTransformFailure(result, "path.reserved-index")
			continue
		}
		data, err := source.Read(scan.manifest.Root, candidate)
		if err != nil {
			return nil, snapshot.Manifest{}, err
		}
		converted, err := transform.Document(candidate.RelativePath, data)
		if err != nil {
			scan.results[index] = addTransformFailure(result, "content.transform-error")
			continue
		}
		outputResult, err := engine.ScanData(ctx, candidate.RelativePath, converted)
		if err != nil {
			return nil, snapshot.Manifest{}, err
		}
		if blocking := scan.allowlist.UnapprovedOutputBlocking(outputResult, scan.scannedAt); len(blocking) != 0 {
			scan.results[index] = addTransformFailure(result, "content.output-safety-mismatch")
			continue
		}
		documents[candidate.RelativePath] = converted
		manifestDocuments = append(manifestDocuments, snapshot.ManifestDocument{
			Path: candidate.RelativePath, SourceSHA256: candidate.SHA256, OutputSHA256: digest(converted),
			ApprovedRules: approvedOutputRules(outputResult, scan.allowlist, scan.scannedAt),
		})
	}

	files := make([]snapshot.File, 0, len(documents)+8)
	for relativePath, data := range documents {
		files = append(files, snapshot.File{Path: relativePath, Data: data})
	}
	indexes, err := buildIndexes(scan.manifest.Candidates)
	if err != nil {
		return nil, snapshot.Manifest{}, err
	}
	manifestIndexes := make([]snapshot.ManifestFile, 0, len(indexes))
	for relativePath, data := range indexes {
		files = append(files, snapshot.File{Path: relativePath, Data: data})
		manifestIndexes = append(manifestIndexes, snapshot.ManifestFile{Path: relativePath, SHA256: digest(data)})
	}
	if err := validateRulesetUnchanged(options.projectRoot, scan.rulesetSHA256); err != nil {
		return nil, snapshot.Manifest{}, err
	}
	contentManifest := snapshot.Manifest{
		Version: snapshot.ManifestVersion, GitleaksVersion: scanner.RequiredGitleaksVersion,
		RulesetSHA256: scan.rulesetSHA256, Documents: manifestDocuments, GeneratedIndexes: manifestIndexes,
	}
	manifestBytes, err := snapshot.EncodeManifest(contentManifest)
	if err != nil {
		return nil, snapshot.Manifest{}, err
	}
	files = append(files, snapshot.File{Path: snapshot.ManifestPath, Data: manifestBytes})
	return files, contentManifest, nil
}

func buildIndexes(candidates []source.Candidate) (map[string][]byte, error) {
	directories := map[string]struct{}{".": {}}
	for _, candidate := range candidates {
		for directory := path.Dir(candidate.RelativePath); directory != "."; directory = path.Dir(directory) {
			directories[directory] = struct{}{}
		}
	}
	indexes := make(map[string][]byte, len(directories))
	for directory := range directories {
		data, err := transform.SectionIndex(directory)
		if err != nil {
			return nil, err
		}
		indexPath := "_index.md"
		if directory != "." {
			indexPath = path.Join(directory, "_index.md")
		}
		indexes[indexPath] = data
	}
	return indexes, nil
}

func addTransformFailure(result scanner.Result, rule string) scanner.Result {
	result.Findings = append(result.Findings, scanner.Finding{
		RuleID: rule, Level: scanner.LevelBlock, RelativePath: result.RelativePath,
		Line: 0, Reason: "内容无法安全转换", Approvable: false,
	})
	return result
}

func approvedOutputRules(result scanner.Result, allowlist approval.Allowlist, at time.Time) []string {
	set := make(map[string]struct{})
	for _, finding := range result.Findings {
		if allowlist.AllowsOutput(result, finding, at) {
			set[finding.RuleID] = struct{}{}
		}
	}
	rules := make([]string, 0, len(set))
	for rule := range set {
		rules = append(rules, rule)
	}
	sort.Strings(rules)
	return rules
}

func rulesetFingerprint(projectRoot string) (string, error) {
	return snapshot.ConfigFingerprint(projectRoot,
		".gitleaks.toml",
		approval.ConfigPath,
		"cmd/contentctl/main.go",
		"config/versions.env",
		"internal/approval/approval.go",
		"internal/scanner/gitleaks.go",
		"internal/scanner/scanner.go",
		"internal/source/source.go",
		"internal/snapshot/manifest.go",
		"internal/transform/transform.go",
	)
}

func validateRulesetUnchanged(projectRoot, expected string) error {
	current, err := rulesetFingerprint(projectRoot)
	if err != nil {
		return err
	}
	if current != expected {
		return errors.New("扫描规则、工具版本或批准配置在执行期间发生变化")
	}
	return nil
}

func pairOutputFinding(sourceResult scanner.Result, selected scanner.Finding, outputResult scanner.Result) (scanner.Finding, error) {
	sourceMatches := make([]scanner.Finding, 0)
	outputMatches := make([]scanner.Finding, 0)
	for _, finding := range sourceResult.Findings {
		if sameFindingKind(finding, selected) {
			sourceMatches = append(sourceMatches, finding)
		}
	}
	for _, finding := range outputResult.Findings {
		if sameFindingKind(finding, selected) {
			outputMatches = append(outputMatches, finding)
		}
	}
	selectedIndex := -1
	for index, finding := range sourceMatches {
		if approval.FindingFingerprint(finding) == approval.FindingFingerprint(selected) {
			selectedIndex = index
			break
		}
	}
	if selectedIndex < 0 || len(sourceMatches) != len(outputMatches) || selectedIndex >= len(outputMatches) {
		return scanner.Finding{}, errors.New("转换前后的实际扫描 finding 无法一一对应, 不可批准")
	}
	return outputMatches[selectedIndex], nil
}

func sameFindingKind(left, right scanner.Finding) bool {
	return left.RuleID == right.RuleID && left.Level == right.Level &&
		left.Reason == right.Reason && left.Approvable == right.Approvable
}

func verifyContentBoundary(contentRoot string, manifest snapshot.Manifest) error {
	expectedFiles := map[string]struct{}{
		"_index.md":                              {},
		path.Join("docs", snapshot.ManifestPath): {},
	}
	for _, document := range manifest.Documents {
		expectedFiles[path.Join("docs", document.Path)] = struct{}{}
	}
	for _, index := range manifest.GeneratedIndexes {
		expectedFiles[path.Join("docs", index.Path)] = struct{}{}
	}
	expectedDirectories := map[string]struct{}{".": {}, "docs": {}}
	for filename := range expectedFiles {
		for directory := path.Dir(filename); directory != "."; directory = path.Dir(directory) {
			expectedDirectories[directory] = struct{}{}
		}
	}
	seenFiles := make(map[string]struct{}, len(expectedFiles))
	err := filepath.WalkDir(contentRoot, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(contentRoot, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("content 中禁止符号链接: %s", relative)
		}
		if entry.IsDir() {
			if _, ok := expectedDirectories[relative]; !ok {
				return fmt.Errorf("content 中存在未受控目录: %s", relative)
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("content 中存在非普通文件: %s", relative)
		}
		if _, ok := expectedFiles[relative]; !ok {
			return fmt.Errorf("content 中存在未受控文件: %s", relative)
		}
		seenFiles[relative] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}
	if len(seenFiles) != len(expectedFiles) {
		return errors.New("content 允许文件集合不完整")
	}
	return nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func validateManifestUnchanged(original source.Manifest) error {
	for _, candidate := range original.Candidates {
		if err := source.Validate(original.Root, candidate); err != nil {
			return err
		}
	}
	current, err := source.Discover(original.Root)
	if err != nil {
		return err
	}
	if len(current.Candidates) != len(original.Candidates) {
		return errors.New("扫描期间 Markdown 候选数量发生变化")
	}
	for index := range original.Candidates {
		before, after := original.Candidates[index], current.Candidates[index]
		if before.RelativePath != after.RelativePath || before.SHA256 != after.SHA256 || before.State != after.State {
			return errors.New("扫描期间 Markdown 候选身份或内容发生变化")
		}
	}
	return nil
}

func counts(results []scanner.Result, allowlist approval.Allowlist, at time.Time) (passed, excluded, warnings int) {
	for _, result := range results {
		if len(allowlist.UnapprovedBlocking(result, at)) == 0 {
			passed++
		} else {
			excluded++
		}
		for _, finding := range result.Findings {
			if finding.Level == scanner.LevelWarning {
				warnings++
			}
		}
	}
	return passed, excluded, warnings
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
