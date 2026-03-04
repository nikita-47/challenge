package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// ─── Terminal size ────────────────────────────────────────────────────────────

type winsize struct {
	Row, Col, Xpixel, Ypixel uint16
}

func termSize() (w, h int) {
	ws := &winsize{}
	syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(os.Stdout.Fd()),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)
	w, h = int(ws.Col), int(ws.Row)
	if w < 80 {
		w = 80
	}
	if h < 24 {
		h = 24
	}
	return
}

// ─── Panel ────────────────────────────────────────────────────────────────────

type panel struct {
	title   string
	color   string
	r0, c0  int             // top-left of content area (1-indexed)
	w, h    int             // content dimensions
	cr, cc  int             // draw cursor within content (0-indexed)
	lines   []string        // committed lines (used for scrolling)
	curLine strings.Builder // line currently being written
	buf     strings.Builder // full raw text (for full-screen view)
}

// ─── Split screen ─────────────────────────────────────────────────────────────

type splitScreen struct {
	mu         sync.Mutex
	panels     [4]*panel
	panelCount int
	termW      int
	half       int
	panelH     int
	midRow     int
	questR     int
	sepR       int
	statusR    int
	question   string
	doneCount  int
}

func newSplitScreen(question string) *splitScreen {
	w, h := termSize()
	half := w / 2

	// Layout rows (1-indexed):
	//   1            top border
	//   2..pH+1      top panel content    (panelH rows)
	//   pH+2         mid border
	//   pH+3..2pH+2  bottom panel content (panelH rows)
	//   2pH+3        bottom border
	//   2pH+4        question line 1
	//   2pH+5        question line 2
	//   2pH+6        thin separator
	//   2pH+7        status
	// => panelH = (h - 7) / 2
	panelH := (h - 7) / 2
	if panelH < 3 {
		panelH = 3
	}
	midRow  := panelH + 2
	questR  := 2*panelH + 4
	sepR    := 2*panelH + 6
	statusR := 2*panelH + 7

	panels := [4]*panel{
		{title: "1. Direct",         color: "\033[94m", r0: 2,          c0: 2,        w: half - 1,     h: panelH},
		{title: "2. Step-by-step",   color: "\033[92m", r0: 2,          c0: half + 2, w: w - half - 2, h: panelH},
		{title: "3. Meta-prompting", color: "\033[93m", r0: midRow + 1, c0: 2,        w: half - 1,     h: panelH},
		{title: "4. Expert panel",   color: "\033[95m", r0: midRow + 1, c0: half + 2, w: w - half - 2, h: panelH},
	}

	ss := &splitScreen{
		panels: panels, panelCount: 4, termW: w, half: half, panelH: panelH,
		midRow: midRow, questR: questR, sepR: sepR, statusR: statusR,
		question: question,
	}

	fmt.Print("\033[2J\033[H\033[?25l")
	ss.drawBorders()

	ss.drawQuestion()
	fmt.Printf("\033[%d;1H%s", sepR, strings.Repeat("─", w))
	fmt.Printf("\033[%d;1HStreaming... (Ctrl+C — отменить)", statusR)

	return ss
}

func (ss *splitScreen) drawBorders() {
	w, half, panelH, midRow := ss.termW, ss.half, ss.panelH, ss.midRow
	hL := strings.Repeat("─", half-1)
	hR := strings.Repeat("─", w-half-2)

	fmt.Printf("\033[1;1H┌%s┬%s┐", hL, hR)
	for r := 2; r <= panelH+1; r++ {
		fmt.Printf("\033[%d;1H│\033[%d;%dH│\033[%d;%dH│", r, r, half+1, r, w)
	}
	fmt.Printf("\033[%d;1H├%s┼%s┤", midRow, hL, hR)
	for r := midRow + 1; r <= midRow+panelH; r++ {
		fmt.Printf("\033[%d;1H│\033[%d;%dH│\033[%d;%dH│", r, r, half+1, r, w)
	}
	fmt.Printf("\033[%d;1H└%s┴%s┘", midRow+panelH+1, hL, hR)

	titleRows := [2]int{1, midRow}
	titleCols := [2]int{3, half + 3}
	for i, p := range ss.panels {
		fmt.Printf("\033[%d;%dH%s %s \033[0m", titleRows[i/2], titleCols[i%2], p.color, p.title)
	}
}

// drawQuestion renders the question across up to 2 lines in the question area.
func (ss *splitScreen) drawQuestion() {
	const prefix = "Вопрос: "
	prefixW := len([]rune(prefix))
	qRunes := []rune(ss.question)
	w := ss.termW
	blank := strings.Repeat(" ", w)
	fmt.Printf("\033[%d;1H%s", ss.questR, blank)
	fmt.Printf("\033[%d;1H%s", ss.questR+1, blank)
	line1Cap := w - prefixW
	if len(qRunes) <= line1Cap {
		fmt.Printf("\033[%d;1H%s%s", ss.questR, prefix, ss.question)
	} else {
		fmt.Printf("\033[%d;1H%s%s", ss.questR, prefix, string(qRunes[:line1Cap]))
		rest := qRunes[line1Cap:]
		indent := strings.Repeat(" ", prefixW)
		line2Cap := w - prefixW
		if len(rest) > line2Cap {
			rest = append(rest[:line2Cap-3], '.', '.', '.')
		}
		fmt.Printf("\033[%d;1H%s%s", ss.questR+1, indent, string(rest))
	}
}

// writeInto is the core write logic. Caller must hold mu (or be single-threaded).
func (ss *splitScreen) writeInto(p *panel, text string, out *strings.Builder) {
	p.buf.WriteString(text)
	for _, ch := range text {
		switch ch {
		case '\r':
			// skip
		case '\n':
			ss.commitLine(p, out)
		default:
			p.curLine.WriteRune(ch)
			fmt.Fprintf(out, "\033[%d;%dH%c", p.r0+p.cr, p.c0+p.cc, ch)
			p.cc++
			if p.cc >= p.w {
				ss.commitLine(p, out)
			}
		}
	}
}

// commitLine moves curLine into p.lines and advances or scrolls the panel.
func (ss *splitScreen) commitLine(p *panel, out *strings.Builder) {
	p.lines = append(p.lines, p.curLine.String())
	p.curLine.Reset()
	if len(p.lines) < p.h {
		// Fast path: still within panel height.
		p.cr++
		p.cc = 0
	} else {
		// Panel full — scroll: show last (h-1) committed lines + blank bottom row for new content.
		blank := strings.Repeat(" ", p.w)
		for r := 0; r < p.h; r++ {
			fmt.Fprintf(out, "\033[%d;%dH%s", p.r0+r, p.c0, blank)
		}
		start := len(p.lines) - (p.h - 1)
		for i, line := range p.lines[start:] {
			runes := []rune(line)
			if len(runes) > p.w {
				runes = runes[:p.w]
			}
			fmt.Fprintf(out, "\033[%d;%dH%s", p.r0+i, p.c0, string(runes))
		}
		p.cr = p.h - 1
		p.cc = 0
	}
}

// write appends text to a panel region. Thread-safe.
func (ss *splitScreen) write(p *panel, text string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	var out strings.Builder
	ss.writeInto(p, text, &out)
	fmt.Fprintf(&out, "\033[%d;1H", ss.statusR)
	fmt.Print(out.String())
}

func (ss *splitScreen) setStatus(text string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	fmt.Printf("\033[%d;1H\033[2K%s", ss.statusR, text)
}

func (ss *splitScreen) markDone() {
	ss.mu.Lock()
	ss.doneCount++
	n := ss.doneCount
	total := ss.panelCount
	ss.mu.Unlock()
	if n < total {
		ss.setStatus(fmt.Sprintf("Streaming... (%d/%d готово) — Ctrl+C чтобы отменить", n, total))
	}
}

// viewPanel shows a panel's full content in full-screen with markdown rendering.
func (ss *splitScreen) viewPanel(idx int) {
	p := ss.panels[idx]
	w := ss.termW

	fmt.Print("\033[2J\033[H")
	fmt.Printf("%s %s \033[0m\n", p.color, p.title)
	fmt.Println(strings.Repeat("─", w))
	fmt.Println()
	fmt.Print(renderMarkdown(p.buf.String()))
	fmt.Printf("\n\n%s\n\033[2mНажми Enter чтобы вернуться к результатам.\033[0m", strings.Repeat("─", w))
}

// redraw repaints the split screen and replays all panel content.
func (ss *splitScreen) redraw() {
	fmt.Print("\033[2J\033[H\033[?25l")
	ss.drawBorders()

	ss.drawQuestion()
	fmt.Printf("\033[%d;1H%s", ss.sepR, strings.Repeat("─", ss.termW))

	for _, p := range ss.panels {
		content := p.buf.String()
		p.cr, p.cc = 0, 0
		p.lines = nil
		p.curLine.Reset()
		p.buf.Reset()
		var out strings.Builder
		ss.writeInto(p, content, &out)
		fmt.Print(out.String())
	}
	fmt.Printf("\033[%d;1H", ss.statusR)
}

func (ss *splitScreen) cleanup() {
	fmt.Print("\033[?25h")
}

// ─── API streaming to panels ──────────────────────────────────────────────────

func streamToPanel(ctx context.Context, apiKey string, cfg config, msgs []message, ss *splitScreen, p *panel) (string, error) {
	body, _ := json.Marshal(buildRequest(cfg, msgs))

	if cfg.verbose {
		ss.write(p, formatCurl(apiKey, body)+"\n")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		ss.write(p, "Error: "+err.Error())
		return "", err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() == nil {
			ss.write(p, "Error: "+err.Error())
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		ss.write(p, fmt.Sprintf("API error (%d)", resp.StatusCode))
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, b)
	}

	return readStreamToPanel(ctx, resp.Body, ss, p)
}

func readStreamToPanel(ctx context.Context, r io.Reader, ss *splitScreen, p *panel) (string, error) {
	var full strings.Builder
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			ss.write(p, event.Delta.Text)
			full.WriteString(event.Delta.Text)
		}
	}

	return full.String(), scanner.Err()
}

// ─── Comparison orchestrator ──────────────────────────────────────────────────

func runComparison(apiKey string, cfg config, question string, scanner *bufio.Scanner) {
	ss := newSplitScreen(question)
	defer ss.cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		select {
		case <-sigCh:
			ss.setStatus("Отмена... ожидаем завершения горутин.")
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(sigCh)
	}()

	var wg sync.WaitGroup
	wg.Add(4)

	// 1. Direct — без дополнительных инструкций
	go func() {
		defer wg.Done()
		p := ss.panels[0]
		ss.write(p, "[Промпт]\n"+question+"\n\n")
		streamToPanel(ctx, apiKey, cfg,
			[]message{{Role: "user", Content: question}},
			ss, p)
		ss.markDone()
	}()

	// 2. Step-by-step — пошаговое решение
	go func() {
		defer wg.Done()
		p := ss.panels[1]
		prompt2 := "Реши задачу пошагово:\n\n" + question
		ss.write(p, "[Промпт]\n"+prompt2+"\n\n")
		streamToPanel(ctx, apiKey, cfg,
			[]message{{Role: "user", Content: prompt2}},
			ss, p)
		ss.markDone()
	}()

	// 3. Meta-prompting — два последовательных запроса
	go func() {
		defer wg.Done()
		p := ss.panels[2]
		metaPrompt := "Напиши оптимальный промпт для точного решения этой задачи. Верни только промпт, без пояснений:\n\n" + question
		ss.write(p, "[Промпт]\n"+metaPrompt+"\n\n[Шаг 1] Составляю оптимальный промпт...\n\n")
		generated, err := streamToPanel(ctx, apiKey, cfg,
			[]message{{Role: "user", Content: metaPrompt}},
			ss, p)
		if err == nil && generated != "" && ctx.Err() == nil {
			ss.write(p, "\n\n[Шаг 2] Использую сгенерированный промпт...\n\n")
			streamToPanel(ctx, apiKey, cfg,
				[]message{{Role: "user", Content: generated}},
				ss, p)
		}
		ss.markDone()
	}()

	// 4. Expert panel — группа экспертов
	go func() {
		defer wg.Done()
		p := ss.panels[3]
		expertPrompt := "Ты — группа из трёх экспертов, которые вместе решают задачу:\n" +
			"- Аналитик: опирается на теорию вероятностей и формальные рассуждения\n" +
			"- Математик: выполняет точные вычисления\n" +
			"- Критик: проверяет допущения и верифицирует ответ\n\n" +
			"Каждый эксперт кратко высказывает свою точку зрения, затем группа приходит к единому ответу.\n\n" +
			"Задача: " + question
		ss.write(p, "[Промпт]\n"+expertPrompt+"\n\n")
		streamToPanel(ctx, apiKey, cfg,
			[]message{{Role: "user", Content: expertPrompt}},
			ss, p)
		ss.markDone()
	}()

	wg.Wait()

	wasCancelled := ctx.Err() != nil
	cancel()

	// Navigation loop: 1–4 = full-screen view, Enter = exit
	for {
		msg := "Готово! Введи 1-4 для просмотра панели, Enter для выхода в чат."
		if wasCancelled {
			msg = "Отменено. Введи 1-4 для просмотра панели, Enter для выхода в чат."
		}
		ss.setStatus(msg)
		fmt.Print("\033[?25h")
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			break
		}
		if len(input) == 1 && input[0] >= '1' && input[0] <= '4' {
			ss.viewPanel(int(input[0] - '1'))
			scanner.Scan() // wait for Enter
			ss.redraw()
			fmt.Print("\033[?25l")
		}
	}

	fmt.Print("\033[?25h")
	_, h := termSize()
	fmt.Printf("\033[%d;1H\n", h)
}

// ─── Temperature comparison ──────────────────────────────────────────────────

func newTempScreen(question string) *splitScreen {
	w, h := termSize()
	third := w / 3

	// Layout: 3 columns, single row of panels
	//   row 1            top border
	//   row 2..pH+1      panel content
	//   row pH+2         bottom border
	//   row pH+3         question line 1
	//   row pH+4         question line 2
	//   row pH+5         separator
	//   row pH+6         status
	// => panelH = h - 6
	panelH := h - 6
	if panelH < 3 {
		panelH = 3
	}

	questR := panelH + 3
	sepR := panelH + 5
	statusR := panelH + 6

	panels := [4]*panel{
		{title: "temp=0", color: "\033[94m", r0: 2, c0: 2, w: third - 1, h: panelH},
		{title: "temp=0.7", color: "\033[92m", r0: 2, c0: third + 2, w: third - 1, h: panelH},
		{title: "temp=1.0", color: "\033[93m", r0: 2, c0: 2*third + 2, w: w - 2*third - 2, h: panelH},
		{}, // unused 4th slot
	}

	ss := &splitScreen{
		panels: panels, panelCount: 3, termW: w, half: third, panelH: panelH,
		midRow: 0, questR: questR, sepR: sepR, statusR: statusR,
		question: question,
	}

	fmt.Print("\033[2J\033[H\033[?25l")
	ss.drawTempBorders()

	ss.drawQuestion()
	fmt.Printf("\033[%d;1H%s", sepR, strings.Repeat("─", w))
	fmt.Printf("\033[%d;1HStreaming... (Ctrl+C — отменить)", statusR)

	return ss
}

func (ss *splitScreen) drawTempBorders() {
	w := ss.termW
	third := ss.half
	panelH := ss.panelH

	h1 := strings.Repeat("─", third-1)
	h2 := strings.Repeat("─", third-1)
	h3 := strings.Repeat("─", w-2*third-2)

	// top border
	fmt.Printf("\033[1;1H┌%s┬%s┬%s┐", h1, h2, h3)
	// panel rows
	for r := 2; r <= panelH+1; r++ {
		fmt.Printf("\033[%d;1H│\033[%d;%dH│\033[%d;%dH│\033[%d;%dH│",
			r, r, third+1, r, 2*third+1, r, w)
	}
	// bottom border
	fmt.Printf("\033[%d;1H└%s┴%s┴%s┘", panelH+2, h1, h2, h3)

	// panel titles
	titles := [3]struct{ col int; color, name string }{
		{3, ss.panels[0].color, ss.panels[0].title},
		{third + 3, ss.panels[1].color, ss.panels[1].title},
		{2*third + 3, ss.panels[2].color, ss.panels[2].title},
	}
	for _, t := range titles {
		fmt.Printf("\033[1;%dH%s %s \033[0m", t.col, t.color, t.name)
	}
}

func (ss *splitScreen) redrawTemp() {
	fmt.Print("\033[2J\033[H\033[?25l")
	ss.drawTempBorders()

	ss.drawQuestion()
	fmt.Printf("\033[%d;1H%s", ss.sepR, strings.Repeat("─", ss.termW))

	for i := 0; i < 3; i++ {
		p := ss.panels[i]
		content := p.buf.String()
		p.cr, p.cc = 0, 0
		p.lines = nil
		p.curLine.Reset()
		p.buf.Reset()
		var out strings.Builder
		ss.writeInto(p, content, &out)
		fmt.Print(out.String())
	}
	fmt.Printf("\033[%d;1H", ss.statusR)
}

func runTempComparison(apiKey string, cfg config, question string, scanner *bufio.Scanner) {
	ss := newTempScreen(question)
	defer ss.cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		select {
		case <-sigCh:
			ss.setStatus("Отмена... ожидаем завершения горутин.")
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(sigCh)
	}()

	temps := [3]float64{0, 0.7, 1.0}

	var wg sync.WaitGroup
	wg.Add(3)

	for i := 0; i < 3; i++ {
		go func(idx int) {
			defer wg.Done()
			p := ss.panels[idx]
			tempCfg := cfg
			tempCfg.temperature = temps[idx]
			streamToPanel(ctx, apiKey, tempCfg,
				[]message{{Role: "user", Content: question}},
				ss, p)
			ss.markDone()
		}(i)
	}

	wg.Wait()

	wasCancelled := ctx.Err() != nil
	cancel()

	for {
		msg := "Готово! Введи 1-3 для просмотра панели, Enter для выхода в чат."
		if wasCancelled {
			msg = "Отменено. Введи 1-3 для просмотра панели, Enter для выхода в чат."
		}
		ss.setStatus(msg)
		fmt.Print("\033[?25h")
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			break
		}
		if len(input) == 1 && input[0] >= '1' && input[0] <= '3' {
			ss.viewPanel(int(input[0] - '1'))
			scanner.Scan()
			ss.redrawTemp()
			fmt.Print("\033[?25l")
		}
	}

	fmt.Print("\033[?25h")
	_, h := termSize()
	fmt.Printf("\033[%d;1H\n", h)
}

// ─── Model comparison ────────────────────────────────────────────────────────

type metrics struct {
	model        string
	provider     string
	duration     time.Duration
	inputTokens  int
	outputTokens int
	costIn       float64
	costOut      float64
}

func (m *metrics) totalCost() float64 {
	return float64(m.inputTokens)*m.costIn/1e6 + float64(m.outputTokens)*m.costOut/1e6
}

func newModelScreen(question string) *splitScreen {
	w, h := termSize()
	third := w / 3

	panelH := h - 6
	if panelH < 3 {
		panelH = 3
	}

	questR := panelH + 3
	sepR := panelH + 5
	statusR := panelH + 6

	panels := [4]*panel{
		{title: "Qwen2.5-1.5B (local)", color: "\033[94m", r0: 2, c0: 2, w: third - 1, h: panelH},
		{title: "GPT-4o-mini", color: "\033[92m", r0: 2, c0: third + 2, w: third - 1, h: panelH},
		{title: "Claude Sonnet", color: "\033[93m", r0: 2, c0: 2*third + 2, w: w - 2*third - 2, h: panelH},
		{},
	}

	ss := &splitScreen{
		panels: panels, panelCount: 3, termW: w, half: third, panelH: panelH,
		midRow: 0, questR: questR, sepR: sepR, statusR: statusR,
		question: question,
	}

	fmt.Print("\033[2J\033[H\033[?25l")
	ss.drawModelBorders()

	ss.drawQuestion()
	fmt.Printf("\033[%d;1H%s", sepR, strings.Repeat("─", w))
	fmt.Printf("\033[%d;1HStreaming from 3 models... (Ctrl+C to cancel)", statusR)

	return ss
}

func (ss *splitScreen) drawModelBorders() {
	w := ss.termW
	third := ss.half
	panelH := ss.panelH

	h1 := strings.Repeat("─", third-1)
	h2 := strings.Repeat("─", third-1)
	h3 := strings.Repeat("─", w-2*third-2)

	fmt.Printf("\033[1;1H┌%s┬%s┬%s┐", h1, h2, h3)
	for r := 2; r <= panelH+1; r++ {
		fmt.Printf("\033[%d;1H│\033[%d;%dH│\033[%d;%dH│\033[%d;%dH│",
			r, r, third+1, r, 2*third+1, r, w)
	}
	fmt.Printf("\033[%d;1H└%s┴%s┴%s┘", panelH+2, h1, h2, h3)

	titles := [3]struct{ col int; color, name string }{
		{3, ss.panels[0].color, ss.panels[0].title},
		{third + 3, ss.panels[1].color, ss.panels[1].title},
		{2*third + 3, ss.panels[2].color, ss.panels[2].title},
	}
	for _, t := range titles {
		fmt.Printf("\033[1;%dH%s %s \033[0m", t.col, t.color, t.name)
	}
}

func (ss *splitScreen) redrawModel() {
	fmt.Print("\033[2J\033[H\033[?25l")
	ss.drawModelBorders()

	ss.drawQuestion()
	fmt.Printf("\033[%d;1H%s", ss.sepR, strings.Repeat("─", ss.termW))

	for i := 0; i < 3; i++ {
		p := ss.panels[i]
		content := p.buf.String()
		p.cr, p.cc = 0, 0
		p.lines = nil
		p.curLine.Reset()
		p.buf.Reset()
		var out strings.Builder
		ss.writeInto(p, content, &out)
		fmt.Print(out.String())
	}
	fmt.Printf("\033[%d;1H", ss.statusR)
}

func streamToPanelOpenAI(ctx context.Context, baseURL, apiKey, model string, cfg config, msgs []message, ss *splitScreen, p *panel) (string, *metrics, error) {
	m := &metrics{model: model, costIn: 0, costOut: 0}
	start := time.Now()

	body, _ := json.Marshal(buildOpenAIRequest(model, cfg, msgs))

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		ss.write(p, "Error: "+err.Error())
		return "", m, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() == nil {
			ss.write(p, "Error: "+err.Error())
		}
		m.duration = time.Since(start)
		return "", m, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		ss.write(p, fmt.Sprintf("API error (%d): %s", resp.StatusCode, string(b)))
		m.duration = time.Since(start)
		return "", m, fmt.Errorf("API error %d: %s", resp.StatusCode, b)
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
			text := event.Choices[0].Delta.Content
			ss.write(p, text)
			full.WriteString(text)
		}
		if event.Usage != nil {
			m.inputTokens = event.Usage.PromptTokens
			m.outputTokens = event.Usage.CompletionTokens
		}
	}

	m.duration = time.Since(start)

	// Fallback: estimate output tokens from character count if not reported
	if m.outputTokens == 0 && full.Len() > 0 {
		m.outputTokens = full.Len() / 4
	}

	return full.String(), m, scanner.Err()
}

func streamToPanelAnthropic(ctx context.Context, apiKey string, cfg config, msgs []message, ss *splitScreen, p *panel) (string, *metrics, error) {
	m := &metrics{model: "claude-sonnet-4-5-20250929"}
	start := time.Now()

	body, _ := json.Marshal(buildRequest(cfg, msgs))

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		ss.write(p, "Error: "+err.Error())
		return "", m, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() == nil {
			ss.write(p, "Error: "+err.Error())
		}
		m.duration = time.Since(start)
		return "", m, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		ss.write(p, fmt.Sprintf("API error (%d)", resp.StatusCode))
		m.duration = time.Since(start)
		return "", m, fmt.Errorf("API error %d: %s", resp.StatusCode, b)
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var raw json.RawMessage
		var event struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(data), &raw); err != nil {
			continue
		}
		if err := json.Unmarshal(raw, &event); err != nil {
			continue
		}

		switch event.Type {
		case "message_start":
			var ms struct {
				Message struct {
					Usage struct {
						InputTokens int `json:"input_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			json.Unmarshal(raw, &ms)
			m.inputTokens = ms.Message.Usage.InputTokens

		case "content_block_delta":
			var cbd struct {
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			json.Unmarshal(raw, &cbd)
			if cbd.Delta.Type == "text_delta" {
				ss.write(p, cbd.Delta.Text)
				full.WriteString(cbd.Delta.Text)
			}

		case "message_delta":
			var md struct {
				Usage struct {
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			json.Unmarshal(raw, &md)
			m.outputTokens = md.Usage.OutputTokens
		}
	}

	m.duration = time.Since(start)
	return full.String(), m, scanner.Err()
}

func printComparisonTable(results [3]*metrics) {
	fmt.Println()
	fmt.Println("┌───────────────────────┬──────────┬────────────┬─────────────┬───────────┐")
	fmt.Println("│ Model                 │ Time     │ Tokens I/O │ Cost        │ Provider  │")
	fmt.Println("├───────────────────────┼──────────┼────────────┼─────────────┼───────────┤")
	for _, m := range results {
		if m == nil {
			continue
		}
		name := m.model
		if len(name) > 21 {
			name = name[:21]
		}
		dur := fmt.Sprintf("%.1fs", m.duration.Seconds())
		tokens := fmt.Sprintf("%d/%d", m.inputTokens, m.outputTokens)
		cost := fmt.Sprintf("$%.6f", m.totalCost())
		fmt.Printf("│ %-21s │ %-8s │ %-10s │ %-11s │ %-9s │\n", name, dur, tokens, cost, m.provider)
	}
	fmt.Println("└───────────────────────┴──────────┴────────────┴─────────────┴───────────┘")
	fmt.Println()
}

func runModelComparison(anthropicKey, openaiKey string, cfg config, question string, scanner *bufio.Scanner) {
	ss := newModelScreen(question)
	defer ss.cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		select {
		case <-sigCh:
			ss.setStatus("Cancelling...")
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(sigCh)
	}()

	models := [3]modelInfo{
		{name: "Qwen2.5-1.5B (local)", provider: "Local", baseURL: "http://localhost:1234", model: "qwen2.5-coder-1.5b-instruct", costIn: 0, costOut: 0},
		{name: "GPT-4o-mini", provider: "OpenAI", baseURL: "https://api.openai.com", apiKey: openaiKey, model: "gpt-4o-mini", costIn: 0.15, costOut: 0.60},
		{name: "Claude Sonnet", provider: "Anthropic", apiKey: anthropicKey, model: "claude-sonnet-4-5-20250929", costIn: 3.00, costOut: 15.00},
	}

	var results [3]*metrics
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(3)

	for i := 0; i < 3; i++ {
		go func(idx int) {
			defer wg.Done()
			p := ss.panels[idx]
			mi := models[idx]
			msgs := []message{{Role: "user", Content: question}}

			var m *metrics
			if mi.provider == "Anthropic" {
				_, m, _ = streamToPanelAnthropic(ctx, mi.apiKey, cfg, msgs, ss, p)
			} else {
				_, m, _ = streamToPanelOpenAI(ctx, mi.baseURL, mi.apiKey, mi.model, cfg, msgs, ss, p)
			}

			if m != nil {
				m.model = mi.name
				m.provider = mi.provider
				m.costIn = mi.costIn
				m.costOut = mi.costOut
			}
			mu.Lock()
			results[idx] = m
			mu.Unlock()
			ss.markDone()
		}(i)
	}

	wg.Wait()

	wasCancelled := ctx.Err() != nil
	cancel()

	for {
		msg := "Done! Press 1-3 to view panel, Enter to see comparison table."
		if wasCancelled {
			msg = "Cancelled. Press 1-3 to view panel, Enter to see comparison table."
		}
		ss.setStatus(msg)
		fmt.Print("\033[?25h")
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			break
		}
		if len(input) == 1 && input[0] >= '1' && input[0] <= '3' {
			ss.viewPanel(int(input[0] - '1'))
			scanner.Scan()
			ss.redrawModel()
			fmt.Print("\033[?25l")
		}
	}

	// Show comparison table after exiting split view
	fmt.Print("\033[?25h\033[2J\033[H")
	fmt.Printf("Question: %s\n", question)
	printComparisonTable(results)
	fmt.Println("Press Enter to continue...")
	scanner.Scan()
}
