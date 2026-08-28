package workspace

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	fetchLogCap    = 6
	fetchBarWidth  = 20
	fetchRedrawMin = 100 * time.Millisecond
)

type fetchItem struct {
	name   string
	status string // ok, skip, fail
	detail string
	logs   []string
}

type fetchDisplay struct {
	w          io.Writer
	tty        bool
	total      int
	done       int
	current    string
	logs       []string
	items      []fetchItem
	prev       int
	width      int
	lastRedraw time.Time
}

func newFetchDisplay(w io.Writer, total int) *fetchDisplay {
	d := &fetchDisplay{w: w, tty: writerIsTTY(w), total: total}
	if d.tty {
		if f, ok := w.(*os.File); ok {
			if cols, _, err := term.GetSize(int(f.Fd())); err == nil && cols > 0 {
				d.width = cols
			}
		}
	}
	return d
}

func writerIsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func (d *fetchDisplay) start(name string) {
	d.current = name
	d.logs = nil
	if d.tty {
		d.redraw()
		return
	}
	fmt.Fprintf(d.w, "[%d/%d] fetching %s\n", d.done+1, d.total, name)
}

func (d *fetchDisplay) log(line string) {
	if i := strings.LastIndexByte(line, '\r'); i >= 0 {
		line = line[i+1:]
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	d.logs = append(d.logs, line)
	if len(d.logs) > fetchLogCap {
		d.logs = d.logs[len(d.logs)-fetchLogCap:]
	}
	if d.tty {
		d.redrawThrottled()
		return
	}
	fmt.Fprintln(d.w, "  "+line)
}

func (d *fetchDisplay) finish(name, status, detail string) {
	item := fetchItem{name: name, status: status, detail: detail}
	if status == "fail" && len(d.logs) > 0 {
		item.logs = append([]string(nil), d.logs...)
		if strings.HasPrefix(detail, "exit status ") {
			item.detail = ""
		}
	}
	d.items = append(d.items, item)
	d.done++
	d.current = ""
	d.logs = nil
	if d.tty {
		d.redraw()
		return
	}
	line := fmt.Sprintf("%-4s %s", item.status, item.name)
	if item.detail != "" {
		line += "  (" + item.detail + ")"
	}
	fmt.Fprintln(d.w, line)
}

func (d *fetchDisplay) close() {
	if d.tty {
		d.current = ""
		d.logs = nil
		d.redraw()
	}
}

func (d *fetchDisplay) lines() []string {
	label := ""
	if d.current != "" {
		label = "fetching " + d.current
	}
	out := []string{fmt.Sprintf("%s %d/%d  %s", progressBar(d.done, d.total, fetchBarWidth), d.done, d.total, strings.TrimSpace(label))}
	for _, it := range d.items {
		line := fmt.Sprintf("  %-4s %s", it.status, it.name)
		if it.detail != "" {
			line += "  (" + it.detail + ")"
		}
		out = append(out, line)
		for _, l := range it.logs {
			out = append(out, "       "+l)
		}
	}
	if d.current != "" {
		out = append(out, "  .... "+d.current)
		for _, l := range d.logs {
			out = append(out, "       "+l)
		}
	}
	return out
}

func (d *fetchDisplay) redrawThrottled() {
	now := time.Now()
	if !d.lastRedraw.IsZero() && now.Sub(d.lastRedraw) < fetchRedrawMin {
		return
	}
	d.redraw()
}

func (d *fetchDisplay) redraw() {
	lines := d.lines()
	if d.prev > 0 {
		fmt.Fprintf(d.w, "\033[%dA\r\033[J", d.prev)
	}
	for _, line := range lines {
		fmt.Fprintln(d.w, clipLine(line, d.width))
	}
	d.prev = len(lines)
	d.lastRedraw = time.Now()
}

func clipLine(s string, width int) string {
	if width <= 0 {
		return s
	}
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	runes := []rune(s)
	return string(runes[:width-1]) + "…"
}

func progressBar(done, total, width int) string {
	if width < 2 {
		width = 2
	}
	if total <= 0 {
		return "[" + strings.Repeat(" ", width) + "]"
	}
	filled := done * width / total
	if filled > width {
		filled = width
	}
	if filled == width {
		return "[" + strings.Repeat("=", width) + "]"
	}
	bar := strings.Repeat("=", filled) + ">" + strings.Repeat(" ", width-filled-1)
	return "[" + bar + "]"
}
