package services

import (
	"regexp"
	"strings"
	"time"
)

// LogEvent 是一条可通知的日志事件，Text 可以包含多行堆栈。
type LogEvent struct {
	Time      time.Time
	Text      string
	Truncated bool
}

// MultilineOptions 限制聚合范围，避免未知格式的日志无限合并。
type MultilineOptions struct {
	Enabled            bool
	ContextBeforeLines int
	ContinuePattern    string
	FlushTimeout       time.Duration
	MaxLines           int
	MaxBytes           int
}

type MultilineAssembler struct {
	options      MultilineOptions
	match        func(string) bool
	continueExpr *regexp.Regexp
	context      []string
	lines        []string
	lineBytes    int
	lastTime     time.Time
	eventTime    time.Time
	active       bool
	truncated    bool
}

func NewMultilineAssembler(options MultilineOptions, match func(string) bool) (*MultilineAssembler, error) {
	if options.ContextBeforeLines < 0 {
		options.ContextBeforeLines = 0
	}
	if options.MaxLines < 1 {
		options.MaxLines = 80
	}
	if options.MaxBytes < 1 {
		options.MaxBytes = 16384
	}
	if options.FlushTimeout <= 0 {
		options.FlushTimeout = time.Second
	}

	assembler := &MultilineAssembler{options: options, match: match}
	if options.Enabled && options.ContinuePattern != "" {
		expr, err := regexp.Compile(options.ContinuePattern)
		if err != nil {
			return nil, err
		}
		assembler.continueExpr = expr
	}
	return assembler, nil
}

// Append 输入一行日志；返回值中可能包含一个已完整的事件。
func (a *MultilineAssembler) Append(line LogEvent) []LogEvent {
	if !a.options.Enabled {
		if a.match(line.Text) {
			return []LogEvent{line}
		}
		return nil
	}

	if a.active {
		if a.isContinuation(line.Text) {
			a.lastTime = line.Time
			if !a.addLine(line.Text) {
				return []LogEvent{a.flush()}
			}
			return nil
		}
		result := []LogEvent{a.flush()}
		return append(result, a.appendIdle(line)...)
	}
	return a.appendIdle(line)
}

func (a *MultilineAssembler) appendIdle(line LogEvent) []LogEvent {
	if a.match(line.Text) {
		a.start(line)
		return nil
	}
	a.addContext(line.Text)
	return nil
}

func (a *MultilineAssembler) isContinuation(text string) bool {
	return a.continueExpr != nil && a.continueExpr.MatchString(text)
}

func (a *MultilineAssembler) start(line LogEvent) {
	// 优先保留命中行；上下文只在剩余配额内从近到远取用。
	lineText := truncateUTF8(line.Text, a.options.MaxBytes)
	remainingLines := a.options.MaxLines - 1
	remainingBytes := a.options.MaxBytes - len(lineText)
	selected := make([]string, 0, len(a.context))
	usedBytes := 0
	for i := len(a.context) - 1; i >= 0 && remainingLines > 0; i-- {
		itemBytes := len(a.context[i]) + 1
		if itemBytes+usedBytes > remainingBytes {
			break
		}
		selected = append(selected, a.context[i])
		usedBytes += itemBytes
		remainingLines--
	}
	for left, right := 0, len(selected)-1; left < right; left, right = left+1, right-1 {
		selected[left], selected[right] = selected[right], selected[left]
	}
	a.lines = selected
	a.lineBytes = usedBytes
	a.truncated = len(lineText) < len(line.Text)
	a.addLine(lineText)
	a.active = true
	a.lastTime = line.Time
	a.eventTime = line.Time
	a.context = nil
}

func (a *MultilineAssembler) addLine(text string) bool {
	if len(a.lines) >= a.options.MaxLines {
		a.truncated = true
		return false
	}
	need := len(text)
	if len(a.lines) > 0 {
		need++
	}
	if a.lineBytes+need > a.options.MaxBytes {
		a.truncated = true
		return false
	}
	a.lines = append(a.lines, text)
	a.lineBytes += need
	return true
}

func (a *MultilineAssembler) addContext(text string) {
	if a.options.ContextBeforeLines == 0 {
		return
	}
	a.context = append(a.context, text)
	if len(a.context) > a.options.ContextBeforeLines {
		a.context = a.context[len(a.context)-a.options.ContextBeforeLines:]
	}
}

func (a *MultilineAssembler) FlushExpired(now time.Time) *LogEvent {
	if !a.active || now.Sub(a.lastTime) < a.options.FlushTimeout {
		return nil
	}
	event := a.flush()
	return &event
}

func (a *MultilineAssembler) Flush() *LogEvent {
	if !a.active {
		return nil
	}
	event := a.flush()
	return &event
}

func (a *MultilineAssembler) flush() LogEvent {
	event := LogEvent{Time: a.eventTime, Text: strings.Join(a.lines, "\n"), Truncated: a.truncated}
	if event.Truncated {
		event.Text = addTruncationSuffix(event.Text, a.options.MaxBytes, "\n[内容已因达到采集上限截断]")
	}
	a.lines = nil
	a.lineBytes = 0
	a.active = false
	a.truncated = false
	return event
}
