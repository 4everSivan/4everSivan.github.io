package transform

import (
	"bytes"
	"testing"
	"time"
)

// testModTime is a fixed instant so front matter assertions stay deterministic.
var testModTime = time.Date(2026, 7, 18, 10, 30, 0, 0, time.FixedZone("CST", 8*60*60))

const testLastmodLine = "lastmod: \"2026-07-18T10:30:00+08:00\"\n"

func TestDocumentUsesFirstH1OutsideFence(t *testing.T) {
	t.Parallel()
	source := []byte("```text\n# not a title\n```\n\n# 真正标题\n\n正文")
	original := bytes.Clone(source)
	got, err := Document("分类/文档.md", source, testModTime)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("---\ntitle: \"真正标题\"\n" + testLastmodLine + "---\n\n```text\n# not a title\n```\n\n正文\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected output:\n%s", got)
	}
	if !bytes.Equal(source, original) {
		t.Fatal("transform modified the source slice")
	}
}

func TestDocumentFallsBackToFilename(t *testing.T) {
	t.Parallel()
	source := []byte("正文\r\n")
	got, err := Document("目录/无标题 文档.md", source, testModTime)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("---\ntitle: \"无标题 文档\"\n" + testLastmodLine + "---\n\n正文\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected filename fallback:\n%s", got)
	}
}

func TestDocumentOmitsLastmodForZeroModTime(t *testing.T) {
	t.Parallel()
	got, err := Document("分类/文档.md", []byte("# 标题\n\n正文\n"), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("---\ntitle: \"标题\"\n---\n\n正文\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("zero modTime must omit lastmod:\n%s", got)
	}
}

func TestDocumentRemovesOnlyExtractedH1AndOneBlankLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "multiple headings",
			body: "前言\n\n# 第一标题\n\n正文\n# 第二标题\n尾声\n",
			want: "---\ntitle: \"第一标题\"\n" + testLastmodLine + "---\n\n前言\n\n正文\n# 第二标题\n尾声\n",
		},
		{
			name: "CRLF and closing markers",
			body: "# 标题 ###\r\n\r\n正文\r\n",
			want: "---\ntitle: \"标题\"\n" + testLastmodLine + "---\n\n正文\r\n",
		},
		{
			name: "hash in title is not a closing marker",
			body: "# C#\n\n正文\n",
			want: "---\ntitle: \"C#\"\n" + testLastmodLine + "---\n\n正文\n",
		},
		{
			name: "heading without final newline",
			body: "正文\n# 标题",
			want: "---\ntitle: \"标题\"\n" + testLastmodLine + "---\n\n正文\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Document("分类/文档.md", []byte(tt.body), testModTime)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, []byte(tt.want)) {
				t.Fatalf("unexpected output:\n%s", got)
			}
		})
	}
}

func TestDocumentRejectsUnsafeInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		body []byte
	}{
		{"escape", "../x.md", []byte("# x")},
		{"wrong extension", "x.MD", []byte("# x")},
		{"hidden", ".x.md", []byte("# x")},
		{"invalid UTF-8", "x.md", []byte{0xff}},
		{"existing front matter", "x.md", []byte("---\ntitle: x\n---\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Document(tt.path, tt.body, testModTime); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestSectionIndex(t *testing.T) {
	t.Parallel()
	got, err := SectionIndex("数据库/PostgreSQL")
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("---\ntitle: \"PostgreSQL\"\n---\n\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
