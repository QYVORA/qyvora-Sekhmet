// Package output renders structured data to the terminal or as machine
// readable JSON/YAML/Markdown/HTML. The terminal renderer is a presentation
// layer only; it is never the source of truth — the underlying models are.
// Every format is fully implemented; there are no stub renderers.
package output

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	yaml "go.yaml.in/yaml/v3"
)

// Format is a supported output format.
type Format string

const (
	FormatTerminal Format = "terminal"
	FormatJSON     Format = "json"
	FormatYAML     Format = "yaml"
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
)

// ParseFormat resolves a user-supplied output format name. The legacy names
// "table" and "text" normalize to "terminal". Unknown formats return a useful
// error rather than being silently accepted.
func ParseFormat(s string) (Format, error) {
	switch Format(strings.ToLower(strings.TrimSpace(s))) {
	case FormatTerminal, "table", "text":
		return FormatTerminal, nil
	case FormatJSON:
		return FormatJSON, nil
	case FormatYAML:
		return FormatYAML, nil
	case FormatMarkdown, "md":
		return FormatMarkdown, nil
	case FormatHTML:
		return FormatHTML, nil
	}
	return "", fmt.Errorf("invalid output format %q: valid values are terminal, json, yaml, markdown, html", s)
}

// Printer renders values in a configured format.
type Printer struct {
	writer io.Writer
	format Format
	color  bool
}

// New returns a terminal printer writing to stdout.
func New() *Printer {
	return &Printer{writer: os.Stdout, format: FormatTerminal, color: true}
}

// SetWriter sets the output writer.
func (p *Printer) SetWriter(w io.Writer) { p.writer = w }

// SetFormat sets the active format.
func (p *Printer) SetFormat(f Format) { p.format = f }

// SetColor toggles ANSI color.
func (p *Printer) SetColor(c bool) { p.color = c }

// Format returns the active format.
func (p *Printer) Format() Format { return p.format }

// Writer returns the underlying output writer.
func (p *Printer) Writer() io.Writer { return p.writer }

// Print renders v in the active format.
func (p *Printer) Print(v any) {
	switch p.format {
	case FormatJSON:
		p.printJSON(v)
	case FormatYAML:
		p.printYAML(v)
	case FormatMarkdown:
		p.printMarkdown(v)
	case FormatHTML:
		p.printHTML(v)
	default:
		_, _ = fmt.Fprintln(p.writer, v)
	}
}

// PrintTable renders a table, or an array of objects in JSON/YAML/Markdown.
// There is no stub: every format renders the same rows.
func (p *Printer) PrintTable(header []string, rows [][]string) {
	switch p.format {
	case FormatJSON:
		p.printJSON(tableEntries(header, rows))
		return
	case FormatYAML:
		p.printYAML(tableEntries(header, rows))
		return
	case FormatMarkdown:
		p.printMarkdownTable(header, rows)
		return
	case FormatHTML:
		p.printHTMLTable(header, rows)
		return
	}
	p.printTerminalTable(header, rows)
}

func tableEntries(header []string, rows [][]string) []map[string]string {
	entries := make([]map[string]string, len(rows))
	for i, row := range rows {
		entry := make(map[string]string)
		for j, h := range header {
			if j < len(row) {
				entry[h] = row[j]
			}
		}
		entries[i] = entry
	}
	return entries
}

func (p *Printer) printTerminalTable(header []string, rows [][]string) {
	colWidths := make([]int, len(header))
	for i, h := range header {
		colWidths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	headerColor := color.New(color.FgWhite, color.Bold)
	altRowColor := color.New(color.FgBlack, color.BgWhite)

	for i, h := range header {
		if i > 0 {
			_, _ = fmt.Fprint(p.writer, "  ")
		}
		_, _ = headerColor.Fprintf(p.writer, "%-*s", colWidths[i], h)
	}
	_, _ = fmt.Fprintln(p.writer)

	for idx, row := range rows {
		for i, cell := range row {
			if i > 0 {
				_, _ = fmt.Fprint(p.writer, "  ")
			}
			if p.color && idx%2 == 1 {
				_, _ = altRowColor.Fprintf(p.writer, "%-*s", colWidths[i], cell)
			} else {
				_, _ = fmt.Fprintf(p.writer, "%-*s", colWidths[i], cell)
			}
		}
		_, _ = fmt.Fprintln(p.writer)
	}
}

func (p *Printer) printMarkdownTable(header []string, rows [][]string) {
	var b strings.Builder
	b.WriteString("| " + strings.Join(header, " | ") + " |\n")
	b.WriteString("| " + strings.Repeat("--- | ", len(header)) + "\n")
	for _, row := range rows {
		row = padRow(header, row)
		b.WriteString("| " + strings.Join(escapeCells(row), " | ") + " |\n")
	}
	_, _ = p.writer.Write([]byte(b.String()))
}

func (p *Printer) printHTMLTable(header []string, rows [][]string) {
	var b strings.Builder
	b.WriteString("<table><thead><tr>")
	for _, h := range header {
		b.WriteString("<th>" + html.EscapeString(h) + "</th>")
	}
	b.WriteString("</tr></thead><tbody>")
	for _, row := range rows {
		row = padRow(header, row)
		b.WriteString("<tr>")
		for _, cell := range row {
			b.WriteString("<td>" + html.EscapeString(cell) + "</td>")
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table>\n")
	_, _ = p.writer.Write([]byte(b.String()))
}

func (p *Printer) printJSON(v any) {
	enc := json.NewEncoder(p.writer)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "json error: %v\n", err)
	}
}

func (p *Printer) printYAML(v any) {
	out, err := yaml.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "yaml error: %v\n", err)
		return
	}
	_, _ = p.writer.Write(out)
}

func (p *Printer) printMarkdown(v any) {
	out, err := yaml.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "markdown error: %v\n", err)
		return
	}
	_, _ = p.writer.Write([]byte("```yaml\n"))
	_, _ = p.writer.Write(out)
	_, _ = p.writer.Write([]byte("```\n"))
}

func (p *Printer) printHTML(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "html error: %v\n", err)
		return
	}
	_, _ = p.writer.Write([]byte("<pre>"))
	_, _ = p.writer.Write([]byte(html.EscapeString(string(data))))
	_, _ = p.writer.Write([]byte("</pre>\n"))
}

func padRow(header []string, row []string) []string {
	if len(row) >= len(header) {
		return row
	}
	row = append([]string(nil), row...)
	for len(row) < len(header) {
		row = append(row, "")
	}
	return row
}

func escapeCells(row []string) []string {
	out := make([]string, len(row))
	for i, cell := range row {
		out[i] = strings.ReplaceAll(strings.ReplaceAll(cell, "|", "\\|"), "\n", "<br>")
	}
	return out
}
