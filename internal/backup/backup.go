package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/biyan113/grok-setup/internal/paths"
)

type Entry struct {
	Name    string
	Path    string
	ModTime time.Time
	Size    int64
}

func EnsureDir(home string) (string, error) {
	dir := paths.BackupDir(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// CopyFile stores a timestamped copy of src under ~/.grok/backups/.
// Returns empty path if src does not exist.
func CopyFile(home, src, prefix string) (string, error) {
	st, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", fmt.Errorf("不是普通文件: %s", src)
	}
	dir, err := EnsureDir(home)
	if err != nil {
		return "", err
	}
	if prefix == "" {
		prefix = filepath.Base(src)
	}
	name := fmt.Sprintf("%s-%s%s",
		strings.TrimSuffix(prefix, filepath.Ext(prefix)),
		time.Now().Format("20060102-150405"),
		filepath.Ext(src),
	)
	dst := filepath.Join(dir, name)

	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	return dst, out.Close()
}

func List(home string) ([]Entry, error) {
	dir := paths.BackupDir(home)
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Entry
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Entry{
			Name:    e.Name(),
			Path:    filepath.Join(dir, e.Name()),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ModTime.After(out[j].ModTime)
	})
	return out, nil
}

func Restore(backupPath, dest string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".restore.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	_ = tmp.Chmod(0o600)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dest)
}
