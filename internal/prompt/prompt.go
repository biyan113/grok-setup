package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/biyan113/grok-setup/internal/paths"
)

type IO struct {
	In     io.Reader
	Out    io.Writer
	Err    io.Writer
	tty    *os.File
	closer io.Closer
}

func New() *IO {
	p := &IO{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
	if f, err := os.Open(paths.TTYPath()); err == nil {
		p.In = f
		p.tty = f
		p.closer = f
	}
	return p
}

func (p *IO) Close() {
	if p.closer != nil {
		_ = p.closer.Close()
	}
}

func (p *IO) Info(format string, args ...any) {
	fmt.Fprintf(p.Err, "ℹ  "+format+"\n", args...)
}

func (p *IO) OK(format string, args ...any) {
	fmt.Fprintf(p.Err, "✓  "+format+"\n", args...)
}

func (p *IO) Warn(format string, args ...any) {
	fmt.Fprintf(p.Err, "!  "+format+"\n", args...)
}

func (p *IO) Errf(format string, args ...any) {
	fmt.Fprintf(p.Err, "✗  "+format+"\n", args...)
}

func (p *IO) Line(prompt string) (string, error) {
	fmt.Fprint(p.Err, prompt)
	sc := bufio.NewScanner(p.In)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return strings.TrimSpace(sc.Text()), nil
}

func (p *IO) Confirm(prompt string, defaultYes bool) (bool, error) {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}
	s, err := p.Line(fmt.Sprintf("?  %s %s ", prompt, hint))
	if err != nil {
		return false, err
	}
	if s == "" {
		return defaultYes, nil
	}
	switch strings.ToLower(s) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func (p *IO) Secret(prompt string) (string, error) {
	fmt.Fprint(p.Err, prompt)
	var fd int
	if p.tty != nil {
		fd = int(p.tty.Fd())
	} else if f, ok := p.In.(*os.File); ok {
		fd = int(f.Fd())
	} else {
		s, err := p.Line("")
		return s, err
	}
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(p.Err)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func (p *IO) Default(prompt, fallback string) (string, error) {
	s, err := p.Line(fmt.Sprintf("%s [默认: %s] ", prompt, fallback))
	if err != nil {
		return "", err
	}
	if s == "" {
		return fallback, nil
	}
	return s, nil
}
