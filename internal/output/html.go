package output

import (
	"fmt"
	"html"
	"reflect"
	"sort"
	"strings"
	"time"
)

// renderHTML serializes v into a real standalone HTML document (semantic
// headings, tables and lists) rather than an escaped JSON <pre> block. Field
// names come from JSON tags when present; maps are sorted for determinism.
func renderHTML(v any) string {
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<title>QYVORA Sekhmet Report</title>\n")
	b.WriteString("<style>body{font-family:system-ui,sans-serif;margin:2rem;line-height:1.5}")
	b.WriteString("h1,h2,h3{margin-bottom:.25rem}table{border-collapse:collapse;margin:.5rem 0 1.5rem}")
	b.WriteString("th,td{border:1px solid #ccc;padding:.35rem .6rem;text-align:left}")
	b.WriteString("th{background:#f4f4f4}code{background:#f4f4f4;padding:.1rem .3rem;border-radius:3px}</style>\n")
	b.WriteString("</head>\n<body>\n")
	b.WriteString("<h1>" + html.EscapeString(kindName(v)) + "</h1>\n")
	writeHTMLValue(&b, "", reflect.ValueOf(v), 1)
	b.WriteString("</body>\n</html>\n")
	return b.String()
}

func writeHTMLValue(b *strings.Builder, name string, val reflect.Value, level int) {
	val = derefValue(val)
	if !val.IsValid() {
		writeHTMLEscaped(b, "*null*")
		return
	}

	switch val.Kind() {
	case reflect.Struct:
		if t, ok := val.Interface().(time.Time); ok {
			writeHTMLEscaped(b, t.UTC().Format(time.RFC3339Nano))
			return
		}
		if name != "" {
			writeHTMLHeading(b, name, level)
		}
		writeHTMLTable(b, [][]any{structEntries(val)})
	case reflect.Map:
		if name != "" {
			writeHTMLHeading(b, name, level)
		}
		if val.Len() == 0 {
			writeHTMLEscaped(b, "(none)")
			return
		}
		for _, e := range mapEntries(val) {
			b.WriteString("<li><strong>" + html.EscapeString(e.key) + "</strong>: " + html.EscapeString(e.fmt) + "</li>\n")
		}
	case reflect.Slice, reflect.Array:
		if val.Len() == 0 {
			writeHTMLEscaped(b, name+" (none)")
			return
		}
		if name != "" {
			writeHTMLHeading(b, name, level)
		}
		if isStructSlice(val) {
			writeHTMLTable(b, sliceEntries(val))
			return
		}
		b.WriteString("<ul>\n")
		for _, item := range htmlList(val) {
			b.WriteString("<li>" + html.EscapeString(item) + "</li>\n")
		}
		b.WriteString("</ul>\n")
	case reflect.Ptr:
		writeHTMLValue(b, name, val.Elem(), level)
	default:
		writeHTMLEscaped(b, fmtValue(val))
	}
}

func writeHTMLHeading(b *strings.Builder, name string, level int) {
	if level > 6 {
		level = 6
	}
	fmt.Fprintf(b, "<h%d>%s</h%d>\n", level, html.EscapeString(name), level)
}

func writeHTMLTable(b *strings.Builder, rows [][]any) {
	if len(rows) == 0 {
		writeHTMLEscaped(b, "(none)")
		return
	}
	headers := headersOf(rows)
	b.WriteString("<table><thead><tr>")
	for _, h := range headers {
		b.WriteString("<th>" + html.EscapeString(h) + "</th>")
	}
	b.WriteString("</tr></thead><tbody>\n")
	for _, row := range rows {
		b.WriteString("<tr>")
		for _, h := range headers {
			cells := ""
			for _, cell := range row {
				if c, ok := cell.([]any); ok && len(c) == 2 && c[0] == h {
					cells = fmt.Sprintf("%v", c[1])
					break
				}
			}
			if cells == "" {
				cells = "-"
			}
			b.WriteString("<td>" + html.EscapeString(cells) + "</td>")
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</tbody></table>\n")
}

func htmlList(val reflect.Value) []string {
	parts := make([]string, 0, val.Len())
	for i := 0; i < val.Len(); i++ {
		e := derefValue(val.Index(i))
		if !e.IsValid() {
			parts = append(parts, "null")
			continue
		}
		if e.Kind() == reflect.Slice || e.Kind() == reflect.Array || e.Kind() == reflect.Map || e.Kind() == reflect.Struct {
			parts = append(parts, "(object)")
			continue
		}
		parts = append(parts, fmt.Sprintf("%v", e.Interface()))
	}
	sort.Strings(parts)
	return parts
}

// writeHTMLEscaped writes text that may span block and inline content.
func writeHTMLEscaped(b *strings.Builder, s string) {
	if strings.HasPrefix(s, "*") && strings.HasSuffix(s, "*") {
		b.WriteString("<em>" + html.EscapeString(strings.Trim(s, "*")) + "</em>\n")
		return
	}
	b.WriteString(html.EscapeString(s) + "\n")
}
