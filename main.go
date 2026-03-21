package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

// === データ構造 ===

// Claude Codeからstdinで渡されるJSON（新旧両方のフォーマットに対応）
type ClaudeCodeInput struct {
	Model struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"model"`

	ContextWindow struct {
		ContextWindowSize   int     `json:"context_window_size"`
		UsedPercentage      float64 `json:"used_percentage"`
		RemainingPercentage float64 `json:"remaining_percentage"`
		CurrentUsage        *struct {
			InputTokens         int `json:"input_tokens"`
			CacheCreationTokens int `json:"cache_creation_input_tokens"`
			CacheReadTokens     int `json:"cache_read_input_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`

	Cost *struct {
		TotalCostUSD      float64 `json:"total_cost_usd"`
		TotalDurationMs   int64   `json:"total_duration_ms"`
		TotalLinesAdded   int     `json:"total_lines_added"`
		TotalLinesRemoved int     `json:"total_lines_removed"`
	} `json:"cost"`

	Workspace *struct {
		CurrentDir string `json:"current_dir"`
		ProjectDir string `json:"project_dir"`
	} `json:"workspace"`

	RateLimits *struct {
		FiveHour *struct {
			UsedPercentage float64 `json:"used_percentage"`
			ResetsAt       int64   `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay *struct {
			UsedPercentage float64 `json:"used_percentage"`
			ResetsAt       int64   `json:"resets_at"`
		} `json:"seven_day"`
	} `json:"rate_limits"`

	SessionID string `json:"session_id"`
	Version   string `json:"version"`

	Vim *struct {
		Mode string `json:"mode"`
	} `json:"vim"`

	Agent *struct {
		Name string `json:"name"`
	} `json:"agent"`
}

// === ヘルパー関数 ===

// Claude Codeの入力を読む
// Windows の bat ラッパーがファイルに保存した JSON を読む
// 引数でファイルパスが渡されればそれを、なければ stdin を読む
func readClaudeCodeInput() (*ClaudeCodeInput, error) {
	debugLog("readClaudeCodeInput called")

	var data []byte
	var err error

	if len(os.Args) > 1 {
		// 引数でファイルパスが渡された場合
		debugLog(fmt.Sprintf("reading from file: %s", os.Args[1]))
		data, err = os.ReadFile(os.Args[1])
	} else {
		// stdin から読む (bash テスト用)
		debugLog("reading from stdin")
		buf := make([]byte, 64*1024)
		n, _ := os.Stdin.Read(buf)
		data = buf[:n]
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	if len(data) == 0 {
		debugLog("no input data")
		return nil, nil
	}

	debugLog(fmt.Sprintf("read %d bytes", len(data)))

	var input ClaudeCodeInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input JSON: %w", err)
	}

	debugLog("parsed successfully")
	return &input, nil
}

// debugLog: ~/.claude/statusline-debug-mode ファイルが存在するときのみログを書き出す
func debugLog(msg string) {
	home := os.Getenv("USERPROFILE")
	if home == "" {
		home = os.Getenv("HOME")
	}
	flagFile := home + "/.claude/statusline-debug-mode"
	if _, err := os.Stat(flagFile); err != nil {
		return // フラグファイルがなければ何もしない
	}
	logPath := home + "/.claude/statusline-debug.log"
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("15:04:05"), msg)
}

// コンテキスト使用率を計算
func calculateContextPercentage(input *ClaudeCodeInput) float64 {
	if input == nil {
		return 0
	}

	if input.ContextWindow.UsedPercentage > 0 {
		if input.ContextWindow.UsedPercentage > 1.0 {
			return input.ContextWindow.UsedPercentage
		}
		return input.ContextWindow.UsedPercentage * 100
	}

	if input.ContextWindow.CurrentUsage != nil && input.ContextWindow.ContextWindowSize > 0 {
		currentTokens := input.ContextWindow.CurrentUsage.InputTokens +
			input.ContextWindow.CurrentUsage.CacheCreationTokens +
			input.ContextWindow.CurrentUsage.CacheReadTokens
		return float64(currentTokens) / float64(input.ContextWindow.ContextWindowSize) * 100
	}

	return 0
}

// モデル名を短縮
func shortenModelName(name string) string {
	switch name {
	case "Claude Opus 4.6":
		return "Opus 4.6"
	case "Claude Opus 4.5":
		return "Opus 4.5"
	case "Claude Sonnet 4.6":
		return "Sonnet 4.6"
	case "Claude Sonnet 4.5":
		return "Sonnet 4.5"
	case "Claude Sonnet 4":
		return "Sonnet 4"
	case "Claude Haiku 4.5":
		return "Haiku 4.5"
	case "Claude Haiku 4":
		return "Haiku 4"
	default:
		trimmed := strings.TrimPrefix(name, "Claude ")
		if len(trimmed) > 12 {
			return trimmed[:12]
		}
		return trimmed
	}
}

// コストをコンパクトにフォーマット
func formatCost(usd float64) string {
	if usd < 0.01 {
		return "$0"
	}
	if usd < 10.0 {
		return fmt.Sprintf("$%.2f", usd)
	}
	if usd < 100.0 {
		return fmt.Sprintf("$%.1f", usd)
	}
	return fmt.Sprintf("$%.0f", usd)
}

// ミリ秒をコンパクトな時間表記にフォーマット
func formatDuration(ms int64) string {
	if ms <= 0 {
		return ""
	}
	d := time.Duration(ms) * time.Millisecond
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return "<1m"
}

// Unixタイムスタンプ(秒)からリセットまでの残り時間をフォーマット
func formatResetTime(resetsAtSec int64) string {
	if resetsAtSec <= 0 {
		return ""
	}
	resetsAt := time.Unix(resetsAtSec, 0)
	remaining := time.Until(resetsAt)
	if remaining <= 0 {
		return "now"
	}
	days := int(remaining.Hours()) / 24
	hours := int(remaining.Hours()) % 24
	minutes := int(remaining.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd%dh%dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return "<1m"
}

// 行数変更をフォーマット
func formatLines(added, removed int) string {
	if added == 0 && removed == 0 {
		return ""
	}
	return fmt.Sprintf("+%d/-%d", added, removed)
}

// === ANSI + Braille バー ===

const (
	ansiReset = "\033[0m"
	ansiDim   = "\033[2m"
)

// braille文字列: 空→フル (8段階)
var brailleChars = []rune{' ', '⣀', '⣄', '⣤', '⣦', '⣶', '⣷', '⣿'}

// パーセンテージに応じた緑→赤グラデーションのANSI色コードを返す
func gradientColor(pct float64) string {
	if pct < 50 {
		r := int(pct * 5.1)
		return fmt.Sprintf("\033[38;2;%d;200;80m", r)
	}
	g := int(200 - (pct-50)*4)
	if g < 0 {
		g = 0
	}
	return fmt.Sprintf("\033[38;2;255;%d;60m", g)
}

// braille文字でプログレスバーを描画
func brailleBar(pct float64, width int) string {
	pct = math.Max(0, math.Min(100, pct))
	level := pct / 100
	var buf strings.Builder
	for i := 0; i < width; i++ {
		segStart := float64(i) / float64(width)
		segEnd := float64(i+1) / float64(width)
		if level >= segEnd {
			buf.WriteRune(brailleChars[7])
		} else if level <= segStart {
			buf.WriteRune(brailleChars[0])
		} else {
			frac := (level - segStart) / (segEnd - segStart)
			idx := int(frac * 7)
			if idx > 7 {
				idx = 7
			}
			buf.WriteRune(brailleChars[idx])
		}
	}
	return buf.String()
}

// ラベル + brailleバー + パーセンテージをフォーマット
func fmtBar(label string, pct float64) string {
	p := int(math.Round(pct))
	return fmt.Sprintf("%s%s%s %s%s%s %d%%", ansiDim, label, ansiReset, gradientColor(pct), brailleBar(pct, 8), ansiReset, p)
}

// ラベル + brailleバー + リセット時間をフォーマット
func fmtBarReset(label string, pct float64, resetsAtSec int64) string {
	rt := formatResetTime(resetsAtSec)
	return fmt.Sprintf("%s%s%s %s%s%s %s", ansiDim, label, ansiReset, gradientColor(pct), brailleBar(pct, 8), ansiReset, rt)
}

func main() {
	ccInput, inputErr := readClaudeCodeInput()
	if inputErr != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", inputErr)
	}

	// パース後のデータをデバッグログに記録
	if ccInput != nil {
		b, _ := json.MarshalIndent(ccInput, "", "  ")
		debugLog("=== parsed ===\n" + string(b))
	} else {
		debugLog("ccInput is nil after parsing")
	}

	if ccInput == nil {
		fmt.Println("Claude Statusline")
		return
	}

	var parts []string

	// モデル名
	if ccInput.Model.DisplayName != "" {
		parts = append(parts, shortenModelName(ccInput.Model.DisplayName))
	}

	// コンテキスト使用量 (brailleバー)
	ctxPct := calculateContextPercentage(ccInput)
	if ctxPct > 0 {
		parts = append(parts, fmtBar("ctx", ctxPct))
	}

	// レートリミット 5時間 (brailleバー)
	if ccInput.RateLimits != nil && ccInput.RateLimits.FiveHour != nil {
		parts = append(parts, fmtBar("5h", ccInput.RateLimits.FiveHour.UsedPercentage))
	}

	// レートリミット 7日 (brailleバー)
	if ccInput.RateLimits != nil && ccInput.RateLimits.SevenDay != nil {
		parts = append(parts, fmtBar("7d", ccInput.RateLimits.SevenDay.UsedPercentage))
	}

	if len(parts) == 0 {
		fmt.Println("Claude Statusline")
		return
	}

	sep := fmt.Sprintf(" %s│%s ", ansiDim, ansiReset)
	fmt.Println(strings.Join(parts, sep))

	// 2段目: レートリミット バー + リセット時間
	var line2Parts []string
	if ccInput.RateLimits != nil && ccInput.RateLimits.FiveHour != nil && ccInput.RateLimits.FiveHour.ResetsAt > 0 {
		line2Parts = append(line2Parts, fmtBarReset("5h", progressPercent(ccInput.RateLimits.FiveHour.ResetsAt, 5*3600), ccInput.RateLimits.FiveHour.ResetsAt))
	}
	if ccInput.RateLimits != nil && ccInput.RateLimits.SevenDay != nil && ccInput.RateLimits.SevenDay.ResetsAt > 0 {
		line2Parts = append(line2Parts, fmtBarReset("7d", progressPercent(ccInput.RateLimits.SevenDay.ResetsAt, 7*24*3600), ccInput.RateLimits.SevenDay.ResetsAt))
	}
	if len(line2Parts) > 0 {
		fmt.Println(strings.Join(line2Parts, sep))
	}
}

func progressPercent(end int64, durationSec int64) float64 {
	start := end - durationSec
	now := time.Now().Unix()
	if now <= start {
		return 0.0
	}
	if now >= end {
		return 100.0
	}
	return float64(now-start) / float64(end-start) * 100.0
}
