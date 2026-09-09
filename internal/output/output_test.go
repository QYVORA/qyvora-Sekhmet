package output

import (
	"strings"
	"testing"
)

func renderFormat(f Format, v any, table bool) string {
	var buf strings.Builder
	p := New()
	p.SetFormat(f)
	p.SetWriter(&buf)
	if table {
		p.PrintTable([]string{"name", "value"}, [][]string{{"a|b", "x<y"}, {"plain", "ok"}})
	} else {
		p.Print(v)
	}
	return buf.String()
}

func TestPrintMarkdownTable(t *testing.T) {
	out := renderFormat(FormatMarkdown, nil, true)
	for _, want := range []string{"| name |", "| a\\|b |", "--- | --- |"} {
		if !strings.Contains(out, want) {
			t.Fatalf("markdown table missing %q:\n%s", want, out)
		}
	}
}

func TestPrintHTMLTable(t *testing.T) {
	out := renderFormat(FormatHTML, nil, true)
	for _, want := range []string{"<table>", "<th>name</th>", "&lt;", "a|b"} {
		if !strings.Contains(out, want) {
			t.Fatalf("html table missing %q:\n%s", want, out)
		}
	}
}

func TestPrintMarkdownValue(t *testing.T) {
	out := renderFormat(FormatMarkdown, map[string]any{"rule": "ADM-001"}, false)
	if !strings.HasPrefix(out, "```yaml") || !strings.Contains(out, "ADM-001") || !strings.HasSuffix(out, "```\n") {
		t.Fatalf("markdown value malformed:\n%s", out)
	}
}

func TestPrintHTMLValue(t *testing.T) {
	out := renderFormat(FormatHTML, []string{"<script>"}, false)
	if !strings.HasPrefix(out, "<pre>") || !strings.HasSuffix(out, "</pre>\n") {
		t.Fatalf("html value malformed:\n%s", out)
	}
	if strings.Contains(out, "<script>") {
		t.Fatalf("html value leaked unescaped markup:\n%s", out)
	}
}

func TestEveryFormatRenders(t *testing.T) {
	for _, f := range []Format{FormatTerminal, FormatJSON, FormatYAML, FormatMarkdown, FormatHTML} {
		table := renderFormat(f, nil, true)
		value := renderFormat(f, map[string]any{"id": "x"}, false)
		if table == "" || value == "" {
			t.Fatalf("format %q produced empty output", f)
		}
	}
}

func TestParseFormatAliases(t *testing.T) {
	cases := map[string]Format{
		"terminal": FormatTerminal, "table": FormatTerminal, "text": FormatTerminal,
		"json": FormatJSON, "yaml": FormatYAML, "md": FormatMarkdown, "markdown": FormatMarkdown, "html": FormatHTML,
	}
	for in, want := range cases {
		got, err := ParseFormat(in)
		if err != nil {
			t.Fatalf("ParseFormat(%q): %v", in, err)
		}
		if got != want {
			t.Fatalf("ParseFormat(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := ParseFormat("bogus"); err == nil {
		t.Fatal("ParseFormat(\"bogus\") should fail")
	}
}

func TestPrintTablePadsShortRows(t *testing.T) {
	var buf strings.Builder
	p := New()
	p.SetFormat(FormatMarkdown)
	p.SetWriter(&buf)
	p.PrintTable([]string{"a", "b", "c"}, [][]string{{"1"}})
	if !strings.Contains(buf.String(), "||  |") && !strings.Contains(buf.String(), "| 1 |  |  |") {
		t.Fatalf("short row not padded:\n%s", buf.String())
	}
}
