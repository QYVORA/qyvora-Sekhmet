package output

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

// renderMarkdown serializes v into real Markdown (headings, lists and tables),
// not a YAML or JSON fence. Field names come from JSON tags when present;
// maps are sorted so output is deterministic.
func renderMarkdown(v any) string {
	var b strings.Builder
	writeMarkdownValue(&b, kindName(v), reflect.ValueOf(v), 0)
	return strings.TrimRight(b.String(), "\n") + "\n"
}

const listIndent = "  "

func writeMarkdownValue(b *strings.Builder, name string, val reflect.Value, depth int) {
	val = derefValue(val)
	if !val.IsValid() {
		writeMDBullet(b, name, *new(string), depth)
		writeMDNode(b, "`null`", depth)
		return
	}

	switch val.Kind() {
	case reflect.Struct:
		if t, ok := val.Interface().(time.Time); ok {
			writeMDBullet(b, name, t.UTC().Format(time.RFC3339Nano), depth)
			return
		}
		writeMDHeading(b, name, depth)
		writeMDTable(b, [][]any{structEntries(val)})
	case reflect.Map:
		writeMDHeading(b, name, depth)
		if val.Len() == 0 {
			writeMDNode(b, "- (none)", depth)
			return
		}
		entries := mapEntries(val)
		for _, e := range entries {
			writeMDBullet(b, e.key, e.fmt, depth)
		}
	case reflect.Slice, reflect.Array:
		if val.Len() == 0 {
			writeMDBullet(b, name, "(none)", depth)
			return
		}
		writeMDHeading(b, name, depth)
		if isStructSlice(val) {
			writeMDTable(b, sliceEntries(val))
			return
		}
		writeMDNode(b, mdList(val), depth)
	case reflect.Ptr:
		writeMarkdownValue(b, name, val.Elem(), depth)
	default:
		writeMDBullet(b, name, fmtValue(val), depth)
	}
}

func writeMDHeading(b *strings.Builder, name string, depth int) {
	level := 2
	if depth > 0 {
		level = 3
	}
	b.WriteString(strings.Repeat("#", level) + " " + name + "\n")
}

func writeMDBullet(b *strings.Builder, key, value string, depth int) {
	writeMDNode(b, "- **"+key+"**: "+value, depth)
}

func writeMDNode(b *strings.Builder, node string, depth int) {
	if depth > 0 {
		b.WriteString(strings.Repeat(listIndent, depth))
	}
	b.WriteString(node + "\n")
}

func writeMDTable(b *strings.Builder, rows [][]any) {
	if len(rows) == 0 {
		b.WriteString("- (none)\n")
		return
	}
	headers := headersOf(rows)
	b.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	b.WriteString("| " + strings.Join(seps, " | ") + " |\n")
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i, h := range headers {
			for _, cell := range row {
				if c, ok := cell.([]any); ok && len(c) == 2 && c[0] == h {
					cells[i] = cellString(c[1])
					break
				}
			}
			if cells[i] == "" {
				cells[i] = "-"
			}
		}
		b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	}
}

func mdList(val reflect.Value) string {
	parts := make([]string, 0, val.Len())
	for i := 0; i < val.Len(); i++ {
		e := derefValue(val.Index(i))
		if !e.IsValid() {
			parts = append(parts, "null")
			continue
		}
		if e.Kind() == reflect.Slice || e.Kind() == reflect.Array || e.Kind() == reflect.Map || e.Kind() == reflect.Struct {
			parts = append(parts, fmt.Sprintf("(%d entry)", e.Len()))
			continue
		}
		parts = append(parts, fmtValue(e))
	}
	return "- " + strings.Join(parts, "\n"+"- ")
}

// structEntries flattens a struct into a single table row of {name, value}
// cells.
func structEntries(val reflect.Value) []any {
	fields := exportedFields(val)
	cells := make([]any, 0, len(fields))
	for _, f := range fields {
		fv := val.FieldByIndex(f.index)
		if f.skip {
			continue
		}
		cells = append(cells, []any{f.name, fmtValue(derefValue(fv))})
	}
	return cells
}

func sliceEntries(val reflect.Value) [][]any {
	rows := make([][]any, 0, val.Len())
	for i := 0; i < val.Len(); i++ {
		rows = append(rows, structEntries(derefValue(val.Index(i))))
	}
	return rows
}

func isStructSlice(val reflect.Value) bool {
	if val.Len() == 0 {
		return false
	}
	k := derefValue(val.Index(0))
	return k.IsValid() && k.Kind() == reflect.Struct
}

// headersOf computes the union of keys in document order.
func headersOf(rows [][]any) []string {
	seen := map[string]bool{}
	var out []string
	for _, row := range rows {
		for _, cell := range row {
			if c, ok := cell.([]any); ok && len(c) == 2 {
				if k, ok := c[0].(string); ok && !seen[k] {
					seen[k] = true
					out = append(out, k)
				}
			}
		}
	}
	return out
}

func cellString(v any) string {
	return escapeMarkdownText(fmt.Sprintf("%v", v))
}

func escapeMarkdownText(s string) string {
	r := strings.NewReplacer("|", "\\|", "\n", " ", "\r", "")
	return r.Replace(s)
}

// kindName returns a human name for a struct (or a fallback label).
func kindName(v any) string {
	if v == nil {
		return "value"
	}
	rv := reflect.ValueOf(v)
	rv = derefValue(rv)
	if rv.IsValid() && rv.Kind() == reflect.Struct {
		return rv.Type().Name()
	}
	return "value"
}

// fieldInfo describes one JSON-serializable struct field.
type fieldInfo struct {
	name  string
	index []int
	skip  bool
}

func exportedFields(val reflect.Value) []fieldInfo {
	typ := val.Type()
	var out []fieldInfo
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name, skip := jsonFieldName(f)
		out = append(out, fieldInfo{name: name, index: f.Index, skip: skip})
	}
	return out
}

func jsonFieldName(f reflect.StructField) (string, bool) {
	tag := f.Tag.Get("json")
	if tag == "" {
		return humanize(f.Name), false
	}
	parts := strings.Split(tag, ",")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0], false
	}
	if contains(parts, "-") {
		return f.Name, true
	}
	return humanize(f.Name), false
}

func humanize(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func fmtValue(val reflect.Value) string {
	val = derefValue(val)
	if !val.IsValid() {
		return "null"
	}
	if t, ok := val.Interface().(time.Time); ok {
		return t.UTC().Format(time.RFC3339Nano)
	}
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		if val.Type().Elem().Kind() == reflect.Uint8 {
			return fmt.Sprintf("0x%x", val.Bytes())
		}
		parts := make([]string, 0, val.Len())
		for i := 0; i < val.Len(); i++ {
			parts = append(parts, fmtValue(val.Index(i)))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case reflect.Map:
		kv := make([]string, 0, val.Len())
		for _, e := range mapEntries(val) {
			kv = append(kv, e.key+"="+e.fmt)
		}
		return "{" + strings.Join(kv, ", ") + "}"
	case reflect.Struct:
		return "(object)"
	case reflect.String:
		return escapeMarkdownText(val.String())
	case reflect.Bool:
		if val.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", val.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", val.Float())
	default:
		return fmt.Sprintf("%v", val.Interface())
	}
}

func derefValue(val reflect.Value) reflect.Value {
	for val.IsValid() && (val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) {
		if val.IsNil() {
			return reflect.Value{}
		}
		val = val.Elem()
	}
	return val
}

type mapEntry struct {
	key string
	fmt string
}

func mapEntries(val reflect.Value) []mapEntry {
	keys := val.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i].Interface()) < fmt.Sprintf("%v", keys[j].Interface())
	})
	out := make([]mapEntry, 0, len(keys))
	for _, k := range keys {
		out = append(out, mapEntry{key: fmt.Sprintf("%v", k.Interface()), fmt: fmtValue(val.MapIndex(k))})
	}
	return out
}
