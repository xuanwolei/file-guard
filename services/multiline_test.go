package services

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"gopkg.in/ini.v1"
)

func newTestAssembler(t *testing.T, options MultilineOptions) *MultilineAssembler {
	t.Helper()
	assembler, err := NewMultilineAssembler(options, func(text string) bool {
		return strings.Contains(text, "ERROR") || strings.Contains(text, "ValueError")
	})
	if err != nil {
		t.Fatal(err)
	}
	return assembler
}

func TestMultilineAssemblerCollectsContinuation(t *testing.T) {
	assembler := newTestAssembler(t, MultilineOptions{
		Enabled: true, ContinuePattern: "^(\\s+at\\s|\\s*Caused by:)",
		ContextBeforeLines: 3, MaxLines: 10, MaxBytes: 1024,
	})
	now := time.Now()
	if events := assembler.Append(LogEvent{Time: now, Text: "ERROR database failed"}); len(events) != 0 {
		t.Fatalf("不应立即输出事件: %#v", events)
	}
	assembler.Append(LogEvent{Time: now.Add(time.Millisecond), Text: "    at repository.Query(repo.go:12)"})
	events := assembler.Append(LogEvent{Time: now.Add(2 * time.Millisecond), Text: "INFO next request"})
	if len(events) != 1 {
		t.Fatalf("期望输出一条事件，实际 %d", len(events))
	}
	want := "ERROR database failed\n    at repository.Query(repo.go:12)"
	if events[0].Text != want {
		t.Fatalf("事件内容错误\nwant: %q\n got: %q", want, events[0].Text)
	}
}

func TestMultilineAssemblerRetainsPythonContext(t *testing.T) {
	assembler := newTestAssembler(t, MultilineOptions{
		Enabled: true, ContinuePattern: "^\\s+", ContextBeforeLines: 3,
		MaxLines: 10, MaxBytes: 1024,
	})
	now := time.Now()
	assembler.Append(LogEvent{Time: now, Text: "Traceback (most recent call last):"})
	assembler.Append(LogEvent{Time: now, Text: "  File \"app.py\", line 7, in run"})
	assembler.Append(LogEvent{Time: now, Text: "    run_task()"})
	assembler.Append(LogEvent{Time: now, Text: "ValueError: invalid input"})
	event := assembler.Flush()
	if event == nil {
		t.Fatal("期望 Flush 输出 Python 异常")
	}
	if !strings.Contains(event.Text, "Traceback") || !strings.Contains(event.Text, "ValueError") {
		t.Fatalf("缺少前置上下文或命中行: %q", event.Text)
	}
}

func TestMultilineAssemblerFlushAndTruncate(t *testing.T) {
	assembler := newTestAssembler(t, MultilineOptions{
		Enabled: true, ContinuePattern: "^\\s+", FlushTimeout: time.Second,
		MaxLines: 2, MaxBytes: 128,
	})
	now := time.Now()
	assembler.Append(LogEvent{Time: now, Text: "ERROR first"})
	assembler.Append(LogEvent{Time: now.Add(time.Millisecond), Text: "  stack frame 1"})
	events := assembler.Append(LogEvent{Time: now.Add(2 * time.Millisecond), Text: "  stack frame 2"})
	if len(events) != 1 || !events[0].Truncated {
		t.Fatalf("达到行上限后应截断并输出: %#v", events)
	}
	if !strings.Contains(events[0].Text, "达到采集上限") {
		t.Fatalf("截断事件应包含原因: %q", events[0].Text)
	}

	assembler.Append(LogEvent{Time: now, Text: "ERROR timeout"})
	event := assembler.FlushExpired(now.Add(2 * time.Second))
	if event == nil || !strings.Contains(event.Text, "ERROR timeout") {
		t.Fatalf("空闲超时应输出事件: %#v", event)
	}
}

func TestBuildMarkdownRespectsByteLimit(t *testing.T) {
	conf := ini.Empty()
	section, err := conf.NewSection("测试项目")
	if err != nil {
		t.Fatal(err)
	}
	notice := &NoticeContent{
		Path:  "D:/非常长的目录/服务日志.log",
		Event: LogEvent{Time: time.Now(), Text: strings.Repeat("错误😀堆栈\n", 100)},
		Guard: &Guard{Section: section, Config: &Config{NoticeMaxBytes: 256, NoticeReservedBytes: 128}},
	}
	_, content := notice.buildMarkdown()
	if len(content) > 256 {
		t.Fatalf("内容超出限制: %d", len(content))
	}
	if !utf8.ValidString(content) {
		t.Fatalf("内容不是合法 UTF-8: %q", content)
	}
	if !strings.HasSuffix(content, "\n```") {
		t.Fatalf("Markdown 代码块未闭合: %q", content)
	}
}
