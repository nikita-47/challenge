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
	mu        sync.Mutex
	panels    [4]*panel
	termW     int
	half      int
	panelH    int
	midRow    int
	questR    int
	sepR      int
	statusR   int
	question  string
	doneCount int
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
		panels: panels, termW: w, half: half, panelH: panelH,
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
	ss.mu.Unlock()
	if n < 4 {
		ss.setStatus(fmt.Sprintf("Streaming... (%d/4 готово) — Ctrl+C чтобы отменить", n))
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
