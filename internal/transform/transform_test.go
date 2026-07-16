package transform

import (
	"bytes"
	"testing"
)

func TestDocumentUsesFirstH1OutsideFence(t *testing.T) {
	t.Parallel()
	source := []byte("```text\n# not a title\n```\n\n# 真正标题\n\n正文")
	got, err := Document("分类/文档.md", source)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("---\ntitle: \"真正标题\"\n---\n\n```text\n# not a title\n```\n\n# 真正标题\n\n正文\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected output:\n%s", got)
	}
}

func TestDocumentFallsBackToFilename(t *testing.T) {
	t.Parallel()
	got, err := Document("目录/无标题 文档.md", []byte("正文\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("title: \"无标题 文档\"")) {
		t.Fatalf("missing filename title: %s", got)
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
			if _, err := Document(tt.path, tt.body); err == nil {
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
