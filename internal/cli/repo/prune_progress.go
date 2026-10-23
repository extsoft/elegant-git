package repo

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	pruneProgressMin = 1500 * time.Millisecond
	pruneRedrawMin   = 100 * time.Millisecond
)

var pruneSleep = time.Sleep

// pruneProgress is a one-line, erased-on-close status shown while a phase collects candidates.
type pruneProgress struct {
	w          io.Writer
	label      string
	width      int
	total      int
	done       int
	item       string
	started    time.Time
	lastRedraw time.Time
}

func newPruneProgress(w io.Writer, label string) *pruneProgress {
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return nil
	}
	p := &pruneProgress{w: w, label: label, started: time.Now()}
	if cols, _, err := term.GetSize(int(f.Fd())); err == nil && cols > 0 {
		p.width = cols
	}
	p.redraw()
	return p
}

func (p *pruneProgress) setTotal(n int) {
	if p == nil {
		return
	}
	p.total = n
	p.redraw()
}

func (p *pruneProgress) tick(item string) {
	if p == nil {
		return
	}
	p.done++
	p.item = item
	if time.Since(p.lastRedraw) >= pruneRedrawMin || p.done >= p.total {
		p.redraw()
	}
}

func (p *pruneProgress) close() {
	if p == nil {
		return
	}
	p.done = p.total
	p.item = ""
	p.redraw()
	if left := pruneProgressMin - time.Since(p.started); left > 0 {
		pruneSleep(left)
	}
	fmt.Fprint(p.w, "\r\033[2K")
}

func (p *pruneProgress) line() string {
	line := fmt.Sprintf("Scanning %s %d/%d", strings.ToLower(p.label), p.done, p.total)
	if p.item != "" {
		line += " : " + trimName(p.item, pruneNameMaxLen)
	}
	if p.width > 1 && len(line) >= p.width {
		line = line[:p.width-1]
	}
	return line
}

func (p *pruneProgress) redraw() {
	fmt.Fprint(p.w, "\r\033[2K"+p.line())
	p.lastRedraw = time.Now()
}
