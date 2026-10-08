package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dahnutz/procsift-ui/internal/offline"
	"github.com/dahnutz/procsift-ui/internal/report"
	safe "github.com/dahnutz/procsift-ui/internal/sanitize"
)

func clean(s string) string { return safe.ForDisplay(s) }

// oneLine flattens untrusted report text onto a single terminal row.
func oneLine(s string) string {
	s = clean(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\n', '\r', '\t':
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func runeWidth(r rune) int {
	if r == 0 {
		return 0
	}
	if r < 0x1100 {
		return 1
	}
	if (r >= 0x1100 && r <= 0x115F) ||
		(r >= 0x2329 && r <= 0x232A) ||
		(r >= 0x2E80 && r <= 0xA4CF) ||
		(r >= 0xAC00 && r <= 0xD7A3) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFE10 && r <= 0xFE19) ||
		(r >= 0xFE30 && r <= 0xFE6F) ||
		(r >= 0xFF00 && r <= 0xFF60) ||
		(r >= 0xFFE0 && r <= 0xFFE6) ||
		(r >= 0x1F300 && r <= 0x1FAFF) ||
		(r >= 0x20000 && r <= 0x3FFFD) {
		return 2
	}
	return 1
}

func colWidth(s string) int {
	n := 0
	for _, r := range s {
		n += runeWidth(r)
	}
	return n
}

func visibleWidth(s string) int {
	return colWidth(oneLine(clean(s)))
}

// clip shortens s to width display columns without splitting a rune.
func clip(s string, width int) string {
	s = oneLine(s)
	if width <= 0 {
		return ""
	}
	if colWidth(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	var b strings.Builder
	w := 0
	limit := width - 1
	for _, r := range s {
		rw := runeWidth(r)
		if w+rw > limit {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	b.WriteString("…")
	return b.String()
}

func pad(s string, width int) string {
	s = clip(s, width)
	n := colWidth(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

func padLeft(s string, width int) string {
	s = clip(s, width)
	n := colWidth(s)
	if n >= width {
		return s
	}
	return strings.Repeat(" ", width-n) + s
}

func fitASCII(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) > width {
		s = s[:width]
	}
	if !utf8.ValidString(s) {
		s = clip(s, width)
	}
	if len(s) < width {
		s += strings.Repeat(" ", width-len(s))
	}
	return s
}

func wrapText(text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if width < 1 {
		width = 1
	}
	rs := []rune(text)
	var out []string
	for len(rs) > 0 {
		if colWidth(string(rs)) <= width {
			out = append(out, string(rs))
			break
		}
		n := 0
		w := 0
		lastSpace := -1
		for i, r := range rs {
			rw := runeWidth(r)
			if w+rw > width {
				if n == 0 {
					n = 1
				}
				break
			}
			w += rw
			n = i + 1
			if r == ' ' {
				lastSpace = i
			}
		}
		if n < 1 {
			n = 1
		}
		cut := n
		if lastSpace > 0 && lastSpace < n-1 && lastSpace >= n/3 {
			cut = lastSpace
		}
		chunk := strings.TrimRight(string(rs[:cut]), " ")
		if chunk != "" {
			out = append(out, chunk)
		}
		if cut < 1 {
			cut = 1
		}
		rs = rs[cut:]
		for len(rs) > 0 && rs[0] == ' ' {
			rs = rs[1:]
		}
	}
	return out
}

func sevWord(sev string) string {
	switch sev {
	case report.SevHigh:
		return "HIGH"
	case report.SevMedium:
		return "MEDIUM"
	case report.SevLow:
		return "LOW"
	case report.SevInfo:
		return "INFO"
	default:
		w := strings.ToUpper(oneLine(sev))
		if w == "" {
			return "?"
		}
		return w
	}
}

func sevANSI(sev string) string {
	switch sev {
	case report.SevHigh:
		return "\x1b[31m"
	case report.SevMedium:
		return "\x1b[33m"
	case report.SevLow:
		return "\x1b[36m"
	case report.SevInfo:
		return "\x1b[2m"
	default:
		return ""
	}
}

type findingCols struct {
	sev, pid, rule, name, sum int
}

func findingBudget(c findingCols) int {
	n := 2
	started := false
	add := func(w int) {
		if w <= 0 {
			return
		}
		if started {
			n++
		}
		started = true
		n += w
	}
	add(c.sev)
	add(c.pid)
	add(c.rule)
	add(c.name)
	add(c.sum)
	return n
}

func fitFindingCols(width int) findingCols {
	if width <= 2 {
		return findingCols{}
	}
	c := findingCols{sev: 6, pid: 7, rule: 22, name: 16, sum: 8}
	for findingBudget(c) > width {
		switch {
		case c.name > 0:
			c.name--
		case c.pid > 5:
			c.pid--
		case c.rule > 16:
			c.rule--
		case c.sum > 10:
			c.sum--
		case c.rule > 0:
			c.rule--
		case c.sum > 0:
			c.sum--
		case c.pid > 0:
			c.pid--
		case c.sev > 0:
			c.sev--
		default:
			return findingCols{}
		}
	}
	for {
		next := c
		next.sum++
		if findingBudget(next) > width {
			break
		}
		c = next
	}
	return c
}

type procCols struct {
	pid, name, cmd int
}

func procBudget(c procCols) int {
	n := 2
	started := false
	add := func(w int) {
		if w <= 0 {
			return
		}
		if started {
			n++
		}
		started = true
		n += w
	}
	add(c.pid)
	add(c.name)
	add(c.cmd)
	return n
}

func fitProcCols(width int) procCols {
	if width <= 2 {
		return procCols{}
	}
	c := procCols{pid: 7, name: 16, cmd: 8}
	for procBudget(c) > width {
		switch {
		case c.name > 0:
			c.name--
		case c.cmd > 0:
			c.cmd--
		case c.pid > 0:
			c.pid--
		default:
			return procCols{}
		}
	}
	for {
		next := c
		next.cmd++
		if procBudget(next) > width {
			break
		}
		c = next
	}
	return c
}

type colPiece struct {
	text string
	w    int
	left bool
	clip bool
}

func joinPieces(marker string, parts []colPiece) string {
	var b strings.Builder
	b.WriteString(marker)
	any := false
	for _, p := range parts {
		if p.w <= 0 {
			continue
		}
		if any {
			b.WriteByte(' ')
		}
		any = true
		switch {
		case p.clip:
			b.WriteString(clip(p.text, p.w))
		case p.left:
			b.WriteString(padLeft(p.text, p.w))
		default:
			b.WriteString(pad(p.text, p.w))
		}
	}
	return b.String()
}

func findingPlain(cols findingCols, sev, pid, rule, name, sum string) string {
	return joinPieces("  ", []colPiece{
		{text: sev, w: cols.sev},
		{text: pid, w: cols.pid, left: true},
		{text: rule, w: cols.rule},
		{text: name, w: cols.name},
		{text: sum, w: cols.sum, clip: true},
	})
}

func procPlain(cols procCols, pid, name, cmd string) string {
	return joinPieces("  ", []colPiece{
		{text: pid, w: cols.pid, left: true},
		{text: name, w: cols.name},
		{text: cmd, w: cols.cmd, clip: true},
	})
}

func styleSeverity(plain, sev string, sevW int, selected bool, width int) string {
	if colWidth(plain) > width {
		plain = clip(plain, width)
	}
	if selected {
		return "\x1b[7m" + pad(plain, width) + "\x1b[0m"
	}
	color := sevANSI(sev)
	if color == "" || sevW <= 0 || len(plain) < 2+sevW {
		return plain
	}
	field := plain[2 : 2+sevW]
	if len(field) != sevW {
		return plain
	}
	for i := 0; i < len(field); i++ {
		if field[i] >= 0x80 {
			return plain
		}
	}
	return plain[:2] + color + field + "\x1b[0m" + plain[2+sevW:]
}

func findingLine(f report.Finding, selected bool, cols findingCols, width int) string {
	pid := "-"
	if f.PID > 0 {
		pid = fmt.Sprintf("%d", f.PID)
	}
	name := f.Process
	if name == "" {
		name = f.Subject
	}
	plain := findingPlain(cols, fitASCII(sevWord(f.Severity), cols.sev), pid, f.Rule, name, f.Summary)
	if selected && len(plain) >= 2 {
		plain = ">" + plain[1:]
	}
	return styleSeverity(plain, f.Severity, cols.sev, selected, width)
}

func findingHead(cols findingCols) string {
	sum := "SUMMARY"
	if cols.sum < 7 {
		sum = "SUM"
	}
	if cols.sum < 3 {
		sum = ""
	}
	return findingPlain(cols, fitASCII("SEV", cols.sev), "PID", "RULE", "NAME", sum)
}

func procLine(p report.Process, selected bool, cols procCols, width int) string {
	plain := procPlain(cols, fmt.Sprintf("%d", p.PID), p.Name, p.Cmdline)
	if selected && len(plain) >= 2 {
		plain = ">" + plain[1:]
	}
	if colWidth(plain) > width {
		plain = clip(plain, width)
	}
	if selected {
		return "\x1b[7m" + pad(plain, width) + "\x1b[0m"
	}
	return plain
}

func procHead(cols procCols) string {
	cmd := "COMMAND"
	if cols.cmd < 7 {
		cmd = "CMD"
	}
	if cols.cmd < 3 {
		cmd = ""
	}
	return procPlain(cols, "PID", "NAME", cmd)
}

func headerLine(idx *report.IndexedReport, tab ViewTab, sourcePath string, width int) string {
	ver := oneLine(idx.Report.Version)
	if ver == "" {
		ver = "?"
	}
	when := "unknown"
	whenShort := "unknown"
	if !idx.Report.Time.IsZero() {
		when = idx.Report.Time.Local().Format("2006-01-02 15:04:05")
		whenShort = idx.Report.Time.Local().Format("15:04:05")
	}
	host := oneLine(idx.Report.Host.Hostname)
	if host == "" {
		host = "unknown"
	}
	c := idx.Report.HostCounts
	if tab == TabScannerSelf {
		c = idx.Report.ScannerCounts
	}
	counts := fmt.Sprintf("high %d  med %d  low %d  info %d", c.High, c.Medium, c.Low, c.Info)
	compact := fmt.Sprintf("H%d M%d L%d I%d", c.High, c.Medium, c.Low, c.Info)
	gap, gapShort := "", ""
	if n := len(idx.Report.Coverage) + len(idx.Report.Limits); n > 0 {
		gap = fmt.Sprintf("incomplete %d", n)
		gapShort = fmt.Sprintf("inc %d", n)
	}
	file := ""
	if sourcePath != "" {
		file = oneLine(filepath.Base(sourcePath))
	}
	id := "ProcSift " + ver
	candidates := [][]string{
		{id, when, host, counts, gap, file},
		{id, whenShort, host, counts, gap, file},
		{id, whenShort, host, counts, gap},
		{id, whenShort, counts, gap},
		{id, counts, gap},
		{id, compact, gap},
		{id, compact, gapShort},
		{id, compact},
		{compact, gap},
		{compact, gapShort},
		{counts},
		{compact},
	}
	best := compact
	for _, parts := range candidates {
		s := joinHeader(parts...)
		if colWidth(s) <= width {
			return s
		}
		best = s
	}
	return clip(best, width)
}

func joinHeader(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("  ")
		}
		b.WriteString(p)
	}
	return b.String()
}

func tabLine(active ViewTab, width int) string {
	names := []string{"Findings", "Processes", "Coverage", "Scanner self"}
	if width < 52 {
		names = []string{"Findings", "Processes", "Coverage", "Scanner"}
	}
	var plain strings.Builder
	var styled strings.Builder
	for i, name := range names {
		label := " " + name + " "
		if colWidth(plain.String())+colWidth(label) > width {
			break
		}
		plain.WriteString(label)
		if ViewTab(i) == active {
			styled.WriteString("\x1b[7m" + label + "\x1b[0m")
		} else {
			styled.WriteString(label)
		}
	}
	return styled.String()
}

func filterLine(filter string, n, width int) string {
	word := "matches"
	if n == 1 {
		word = "match"
	}
	s := clip(fmt.Sprintf("filter: %s   %d %s", oneLine(filter), n, word), width)
	return "\x1b[33m" + s + "\x1b[0m"
}

func footerLine(width int, status, hint string) string {
	status = oneLine(status)
	hint = oneLine(hint)
	if status == "" {
		return clip(hint, width)
	}
	if hint != "" && colWidth(status)+2+colWidth(hint) <= width {
		return "\x1b[33m" + status + "\x1b[0m  " + hint
	}
	return "\x1b[33m" + clip(status, width) + "\x1b[0m"
}

func browseHint(width int) string {
	full := "↑/↓ PgUp/PgDn   Enter open   p PID   / filter   Tab view   q quit"
	mid := "j/k PgUp/PgDn  Enter  p  /  Tab  q"
	short := "j/k  Enter  p  /  Tab  q"
	if colWidth(full) <= width {
		return full
	}
	if colWidth(mid) <= width {
		return mid
	}
	return short
}

func dim(s string) string {
	if s == "" {
		return ""
	}
	return "\x1b[2m" + s + "\x1b[0m"
}

type canvas struct {
	w, h int
	rows []string
}

func newCanvas(w, h int) *canvas {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return &canvas{w: w, h: h, rows: make([]string, h)}
}

func (c *canvas) put(row int, text string) {
	if row < 0 || row >= c.h {
		return
	}
	if strings.ContainsAny(text, "\n\r") || visibleWidth(text) > c.w {
		text = clip(text, c.w)
	}
	c.rows[row] = text
}

func ensureScroll(cursor, scroll, body int) int {
	if body < 1 {
		return 0
	}
	if cursor < scroll {
		return cursor
	}
	if cursor >= scroll+body {
		return cursor - body + 1
	}
	return scroll
}

func (a *App) clampCursor(n int) {
	if n <= 0 {
		a.cursor = 0
		a.scroll = 0
		return
	}
	if a.cursor < 0 {
		a.cursor = 0
	}
	if a.cursor >= n {
		a.cursor = n - 1
	}
}

func (a *App) renderRows(width, height int) []string {
	switch a.screen {
	case screenFolder:
		return a.renderFolder(width, height)
	case screenFinding:
		return a.renderFinding(width, height)
	case screenPID:
		return a.renderPIDView(width, height)
	default:
		rows := a.renderBrowse(width, height)
		if a.screen == screenPrompt && height > 0 {
			rows[height-1] = a.promptLine(width)
		}
		return rows
	}
}

func (a *App) renderBrowse(width, height int) []string {
	c := newCanvas(width, height)
	if height < 1 {
		return c.rows
	}
	if height == 1 {
		path := ""
		if a.SourcePath != "" {
			path = a.SourcePath
		}
		c.put(0, headerLine(a.Idx, a.tab, path, width))
		return c.rows
	}
	footerAt := height - 1
	row := 0
	pathLine := a.SourcePath != "" && height >= 5
	headerPath := ""
	if a.SourcePath != "" && !pathLine {
		headerPath = a.SourcePath
	}
	c.put(row, headerLine(a.Idx, a.tab, headerPath, width))
	row++
	if pathLine && row < footerAt {
		c.put(row, clip("file  "+a.SourcePath, width))
		row++
	}
	if row < footerAt {
		c.put(row, tabLine(a.tab, width))
		row++
	}
	if a.filter != "" && row < footerAt {
		c.put(row, filterLine(a.filter, a.listLen(), width))
		row++
	}
	fcols := fitFindingCols(width)
	pcols := fitProcCols(width)
	showHead := row < footerAt && height >= 8 && a.tab != TabCoverage
	if showHead {
		switch a.tab {
		case TabFindings, TabScannerSelf:
			c.put(row, dim(findingHead(fcols)))
		case TabProcesses:
			c.put(row, dim(procHead(pcols)))
		}
		row++
	}
	body := footerAt - row
	if body < 0 {
		body = 0
	}
	a.pageRows = body
	switch a.tab {
	case TabFindings, TabScannerSelf:
		a.paintFindings(c, row, body, fcols)
	case TabProcesses:
		a.paintProcesses(c, row, body, pcols)
	default:
		a.paintCoverage(c, row, body)
	}
	c.put(footerAt, footerLine(width, a.statusMsg, browseHint(width)))
	return c.rows
}

func (a *App) paintFindings(c *canvas, start, n int, cols findingCols) {
	list := a.currentFindings()
	a.clampCursor(len(list))
	if n <= 0 {
		return
	}
	if len(list) == 0 {
		msg := "(no findings)"
		if a.filter != "" {
			msg = "(no matches)"
		}
		c.put(start, msg)
		return
	}
	a.scroll = ensureScroll(a.cursor, a.scroll, n)
	for i := 0; i < n; i++ {
		idx := a.scroll + i
		if idx >= len(list) {
			break
		}
		c.put(start+i, findingLine(list[idx], idx == a.cursor, cols, c.w))
	}
}

func (a *App) paintProcesses(c *canvas, start, n int, cols procCols) {
	list := a.filteredProcesses()
	a.clampCursor(len(list))
	if n <= 0 {
		return
	}
	if len(list) == 0 {
		msg := "(no processes recorded)"
		if a.filter != "" {
			msg = "(no matches)"
		}
		c.put(start, msg)
		return
	}
	a.scroll = ensureScroll(a.cursor, a.scroll, n)
	for i := 0; i < n; i++ {
		idx := a.scroll + i
		if idx >= len(list) {
			break
		}
		c.put(start+i, procLine(list[idx], idx == a.cursor, cols, c.w))
	}
}

func (a *App) paintCoverage(c *canvas, start, n int) {
	lines := a.coverageLines()
	a.clampCursor(len(lines))
	if n <= 0 {
		return
	}
	a.scroll = ensureScroll(a.cursor, a.scroll, n)
	for i := 0; i < n; i++ {
		idx := a.scroll + i
		if idx >= len(lines) {
			break
		}
		marker := "  "
		selected := idx == a.cursor
		if selected {
			marker = "> "
		}
		text := marker + oneLine(lines[idx])
		if selected {
			c.put(start+i, "\x1b[7m"+pad(text, c.w)+"\x1b[0m")
		} else {
			c.put(start+i, clip(text, c.w))
		}
	}
}

func (a *App) promptLine(width int) string {
	label := "Filter "
	if a.promptKind == "pid" {
		label = "PID "
	}
	text := label + oneLine(a.promptInput) + "_"
	hint := "Enter confirm   Esc cancel"
	if colWidth(text)+2+colWidth(hint) <= width {
		text += "  " + hint
	}
	return "\x1b[7m" + clip(text, width) + "\x1b[0m"
}

func (a *App) renderFinding(width, height int) []string {
	if a.Idx == nil || a.listIndex < 0 || a.listIndex >= len(a.Idx.AllFindings) {
		return a.renderDetail("Finding", []string{"(missing finding)"}, width, height)
	}
	f := a.Idx.AllFindings[a.listIndex]
	return a.renderDetail(findingTitle(f, width), findingBody(a.Idx, f, width), width, height)
}

func (a *App) renderPIDView(width, height int) []string {
	if a.Idx == nil {
		return a.renderDetail("Process", []string{"(no report)"}, width, height)
	}
	p, ok := a.Idx.ProcessByPID[a.pid]
	title := fmt.Sprintf("PID %d", a.pid)
	if !ok {
		return a.renderDetail(clip(title+"  not in report", width), nil, width, height)
	}
	if p.Name != "" {
		title = fmt.Sprintf("PID %d   %s", a.pid, oneLine(p.Name))
	}
	return a.renderDetail(clip(title, width), pidBody(a.Idx, p, width), width, height)
}

func findingTitle(f report.Finding, width int) string {
	word := sevWord(f.Severity)
	plain := clip(word+"  "+oneLine(f.Rule), width)
	color := sevANSI(f.Severity)
	if color == "" || !strings.HasPrefix(plain, word) {
		return plain
	}
	return color + word + "\x1b[0m" + plain[len(word):]
}

func (a *App) renderDetail(title string, lines []string, width, height int) []string {
	c := newCanvas(width, height)
	if height <= 1 {
		c.put(0, title)
		return c.rows
	}
	c.put(0, title)
	bodyN := height - 2
	if bodyN < 1 {
		a.pageRows = 1
	} else {
		a.pageRows = bodyN
	}
	maxScroll := len(lines) - bodyN
	if maxScroll < 0 {
		maxScroll = 0
	}
	if a.detailScroll > maxScroll {
		a.detailScroll = maxScroll
	}
	if a.detailScroll < 0 {
		a.detailScroll = 0
	}
	for i := 0; i < bodyN; i++ {
		idx := a.detailScroll + i
		if idx >= len(lines) {
			break
		}
		c.put(1+i, lines[idx])
	}
	hint := "Esc back   j/k PgUp/PgDn   q quit"
	if bodyN > 0 && len(lines) > bodyN {
		last := a.detailScroll + bodyN
		if last > len(lines) {
			last = len(lines)
		}
		hint = fmt.Sprintf("%d-%d / %d   PgUp/PgDn   Esc   q", a.detailScroll+1, last, len(lines))
	}
	c.put(height-1, clip(hint, width))
	return c.rows
}

func addKV(lines *[]string, width int, key, val string) {
	if width < 1 {
		return
	}
	val = oneLine(val)
	label := pad(oneLine(key), 10)
	if width < 16 {
		*lines = append(*lines, clip(strings.TrimRight(label, " ")+" "+val, width))
		return
	}
	room := width - 10
	parts := wrapText(val, room)
	if len(parts) == 0 {
		parts = []string{""}
	}
	*lines = append(*lines, clip(label+parts[0], width))
	indent := strings.Repeat(" ", 10)
	for _, p := range parts[1:] {
		*lines = append(*lines, clip(indent+p, width))
	}
}

func addBlock(lines *[]string, width int, key, val string) {
	paras := strings.Split(clean(val), "\n")
	first := true
	for _, para := range paras {
		para = strings.TrimSpace(strings.ReplaceAll(para, "\t", " "))
		if para == "" {
			continue
		}
		k := ""
		if first {
			k = key
			first = false
		}
		addKV(lines, width, k, para)
	}
	if first {
		addKV(lines, width, key, "")
	}
}

func findingBody(idx *report.IndexedReport, f report.Finding, width int) []string {
	var lines []string
	addKV(&lines, width, "Origin", unknownIfEmpty(f.Origin))
	addBlock(&lines, width, "Summary", f.Summary)
	if strings.TrimSpace(f.Detail) != "" {
		addBlock(&lines, width, "Detail", f.Detail)
	}
	if f.PID > 0 {
		addKV(&lines, width, "PID", fmt.Sprintf("%d", f.PID))
	}
	if f.Process != "" {
		addKV(&lines, width, "Process", f.Process)
	}
	if f.Subject != "" {
		addKV(&lines, width, "Subject", f.Subject)
	}
	if len(f.Evidence) == 0 {
		addKV(&lines, width, "Evidence", "(none)")
	} else {
		for i, e := range f.Evidence {
			key := ""
			if i == 0 {
				key = "Evidence"
			}
			addBlock(&lines, width, key, e)
		}
	}
	if f.PID > 0 {
		p, ok := idx.ProcessByPID[f.PID]
		if !ok {
			addKV(&lines, width, "Linked", "not in process table")
		} else {
			addKV(&lines, width, "Linked", unknownIfEmpty(p.Name))
			addKV(&lines, width, "Exe", unknownIfEmpty(p.Exe))
			addKV(&lines, width, "Cmdline", unknownIfEmpty(p.Cmdline))
		}
	}
	return lines
}

func pidBody(idx *report.IndexedReport, p report.Process, width int) []string {
	var lines []string
	addKV(&lines, width, "Boot ID", unknownIfEmpty(idx.Report.Host.BootID))
	addKV(&lines, width, "PPID", fmt.Sprintf("%d", p.PPID))
	addKV(&lines, width, "UID", fmt.Sprintf("%d", p.UID))
	addKV(&lines, width, "State", unknownIfEmpty(p.State))
	addKV(&lines, width, "Threads", fmt.Sprintf("%d", p.Threads))
	addKV(&lines, width, "RSS KiB", fmt.Sprintf("%d", p.RSSKiB))
	addKV(&lines, width, "Name", unknownIfEmpty(p.Name))
	addKV(&lines, width, "Exe", unknownIfEmpty(p.Exe))
	addKV(&lines, width, "Cmdline", unknownIfEmpty(p.Cmdline))
	addKV(&lines, width, "Cwd", unknownIfEmpty(p.Cwd))
	addKV(&lines, width, "Ticks", unknownUint(p.StartTicks))
	addKV(&lines, width, "Exe dev", unknownIfEmpty(p.ExeDev))
	addKV(&lines, width, "Inode", unknownUint(p.ExeInode))
	addKV(&lines, width, "Hash", unknownIfEmpty(p.ExeHash))
	addKV(&lines, width, "Scope", unknownIfEmpty(p.ExeHashScope))
	listed := "false"
	if p.Listed {
		listed = "true"
	}
	addKV(&lines, width, "Listed", listed)
	addKV(&lines, width, "Kthread", unknownBool(p.Kthread, p.KthreadKnown))
	addKV(&lines, width, "Startup", unknownIfEmpty(p.StartupStatus))
	socks := idx.SocketsByPID[p.PID]
	if len(socks) == 0 {
		addKV(&lines, width, "Sockets", "(none recorded)")
	} else {
		for i, s := range socks {
			key := ""
			if i == 0 {
				key = "Sockets"
			}
			val := fmt.Sprintf("%s %s -> %s %s inode %s", s.Proto, emptyDash(s.Local), emptyDash(s.Remote), emptyDash(s.State), unknownUint(s.Inode))
			addKV(&lines, width, key, val)
		}
	}
	findings := idx.FindingsByPID[p.PID]
	if len(findings) == 0 {
		addKV(&lines, width, "Findings", "(none)")
	} else {
		for i, f := range findings {
			key := ""
			if i == 0 {
				key = "Findings"
			}
			addKV(&lines, width, key, sevWord(f.Severity)+"  "+oneLine(f.Rule)+"  "+oneLine(f.Summary))
		}
	}
	return lines
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return oneLine(s)
}

func (a *App) renderFolder(width, height int) []string {
	c := newCanvas(width, height)
	if height < 1 {
		return c.rows
	}
	if height == 1 {
		c.put(0, clip(fmt.Sprintf("Select a report (%d)", len(a.picker)), width))
		return c.rows
	}
	c.put(0, clip(fmt.Sprintf("Select a report   %d json", len(a.picker)), width))
	footerAt := height - 1
	body := footerAt - 1
	if body < 1 {
		body = 0
	}
	a.pageRows = body
	n := len(a.picker)
	if n == 0 {
		a.pickerCursor = 0
		a.scroll = 0
		if body > 0 {
			c.put(1, "(no JSON reports)")
		}
	} else {
		if a.pickerCursor < 0 {
			a.pickerCursor = 0
		}
		if a.pickerCursor >= n {
			a.pickerCursor = n - 1
		}
		a.scroll = ensureScroll(a.pickerCursor, a.scroll, body)
		for i := 0; i < body; i++ {
			idx := a.scroll + i
			if idx >= n {
				break
			}
			c.put(1+i, folderLine(a.picker[idx], idx == a.pickerCursor, width))
		}
	}
	hint := "j/k PgUp/PgDn   Enter open   q quit"
	c.put(footerAt, footerLine(width, a.pickerErr, hint))
	return c.rows
}

func folderLine(e offline.Entry, selected bool, width int) string {
	name := oneLine(filepath.Base(e.Path))
	when := time.Unix(e.ModTime, 0).Local().Format("2006-01-02 15:04")
	marker := "  "
	if selected {
		marker = "> "
	}
	var plain string
	if width >= 40 && colWidth(when)+4 < width {
		room := width - 2 - 1 - colWidth(when)
		if room < 1 {
			room = 1
		}
		plain = marker + pad(name, room) + " " + when
	} else {
		plain = marker + name
	}
	if colWidth(plain) > width {
		plain = clip(plain, width)
	}
	if selected {
		return "\x1b[7m" + pad(plain, width) + "\x1b[0m"
	}
	return plain
}

func matchesFilter(f report.Finding, filter string) bool {
	if filter == "" {
		return true
	}
	filter = strings.ToLower(filter)
	check := func(s string) bool {
		return strings.Contains(strings.ToLower(s), filter)
	}
	return check(f.Rule) || check(f.Summary) || check(f.Detail) || check(f.Process) ||
		check(f.Subject) || check(f.Severity) || check(fmt.Sprintf("%d", f.PID))
}

func matchesProcessFilter(p report.Process, filter string) bool {
	if filter == "" {
		return true
	}
	filter = strings.ToLower(filter)
	check := func(s string) bool {
		return strings.Contains(strings.ToLower(s), filter)
	}
	return check(p.Name) || check(p.Cmdline) || check(p.Exe) || check(fmt.Sprintf("%d", p.PID))
}
