package cfg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/biyan113/grok-setup/internal/secret"
)

// Doc is a generic TOML document. Unknown sections survive a merge.
type Doc map[string]any

func Empty() Doc {
	return Doc{}
}

func Parse(data []byte) (Doc, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return Empty(), nil
	}
	var doc Doc
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("解析 TOML: %w", err)
	}
	if doc == nil {
		doc = Empty()
	}
	return doc, nil
}

func Load(path string) (Doc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Empty(), nil
		}
		return nil, err
	}
	return Parse(data)
}

func Encode(doc Doc) ([]byte, error) {
	if doc == nil {
		doc = Empty()
	}
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(map[string]any(doc)); err != nil {
		return nil, fmt.Errorf("编码 TOML: %w", err)
	}
	return buf.Bytes(), nil
}

// WriteAtomic writes with mode 0600 via a sibling temp file.
func WriteAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config.toml.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o600); err != nil && !chmodUnsupported(err) {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func chmodUnsupported(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not supported")
}

func Clone(doc Doc) Doc {
	return cloneMap(doc)
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneValue(v)
	}
	return out
}

func cloneValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return cloneMap(t)
	case Doc:
		return cloneMap(t)
	case []any:
		cp := make([]any, len(t))
		for i, item := range t {
			cp[i] = cloneValue(item)
		}
		return cp
	default:
		return t
	}
}

// Merge copies src onto dst. Nested tables are merged; other values replace.
func Merge(dst, src Doc) Doc {
	if dst == nil {
		dst = Empty()
	}
	out := cloneMap(dst)
	mergeInto(out, src)
	return out
}

func mergeInto(dst map[string]any, src map[string]any) {
	for k, sv := range src {
		if sv == nil {
			continue
		}
		srcTable, srcIsTable := asTable(sv)
		if existing, ok := dst[k]; ok {
			if dstTable, dstIsTable := asTable(existing); dstIsTable && srcIsTable {
				mergeInto(dstTable, srcTable)
				dst[k] = dstTable
				continue
			}
		}
		dst[k] = cloneValue(sv)
	}
}

func asTable(v any) (map[string]any, bool) {
	switch t := v.(type) {
	case map[string]any:
		return t, true
	case Doc:
		return t, true
	default:
		return nil, false
	}
}

func Table(doc Doc, keys ...string) map[string]any {
	cur := map[string]any(doc)
	for _, key := range keys {
		next, ok := asTable(cur[key])
		if !ok || next == nil {
			next = map[string]any{}
			cur[key] = next
		}
		cur = next
	}
	return cur
}

func GetTable(doc Doc, keys ...string) (map[string]any, bool) {
	cur := map[string]any(doc)
	for _, key := range keys {
		next, ok := asTable(cur[key])
		if !ok || next == nil {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

func StringAt(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func DeleteKey(m map[string]any, key string) {
	if m == nil {
		return
	}
	delete(m, key)
}

func ModelNames(doc Doc) []string {
	models, ok := GetTable(doc, "model")
	if !ok {
		return nil
	}
	names := make([]string, 0, len(models))
	for name, v := range models {
		if _, isTable := asTable(v); isTable {
			names = append(names, name)
		}
	}
	return names
}

// Redact returns a copy with sensitive values masked. Used for show / dry-run.
func Redact(doc Doc) Doc {
	out := cloneMap(doc)
	redactMap(out)
	return out
}

func redactMap(m map[string]any) {
	for k, v := range m {
		if secret.IsSensitiveKey(k) {
			if s, ok := v.(string); ok {
				m[k] = secret.Mask(s)
				continue
			}
		}
		switch t := v.(type) {
		case map[string]any:
			redactMap(t)
		case Doc:
			redactMap(t)
		case []any:
			for _, item := range t {
				if tm, ok := asTable(item); ok {
					redactMap(tm)
				}
			}
		}
	}
}
