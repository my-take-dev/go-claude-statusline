package main

import (
	"encoding/json"
	"fmt"
	"io"
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

// stdinからClaude Codeの入力を読む
func readClaudeCodeInput() (*ClaudeCodeInput, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat stdin: %w", err)
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}

	var input ClaudeCodeInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input JSON: %w", err)
	}

	return &input, nil
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

// 行数変更をフォーマット
func formatLines(added, removed int) string {
	if added == 0 && removed == 0 {
		return ""
	}
	return fmt.Sprintf("+%d/-%d", added, removed)
}

func main() {
	ccInput, inputErr := readClaudeCodeInput()
	if inputErr != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", inputErr)
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

	// コンテキスト使用量
	ctxPct := calculateContextPercentage(ccInput)
	if ctxPct > 0 {
		parts = append(parts, fmt.Sprintf("Ctx: %.0f%%", ctxPct))
	}

	// セッションコスト
	if ccInput.Cost != nil && ccInput.Cost.TotalCostUSD > 0 {
		parts = append(parts, formatCost(ccInput.Cost.TotalCostUSD))
	}

	// セッション時間
	if ccInput.Cost != nil && ccInput.Cost.TotalDurationMs > 0 {
		dur := formatDuration(ccInput.Cost.TotalDurationMs)
		if dur != "" {
			parts = append(parts, dur)
		}
	}

	// 行数変更
	if ccInput.Cost != nil {
		lineStr := formatLines(ccInput.Cost.TotalLinesAdded, ccInput.Cost.TotalLinesRemoved)
		if lineStr != "" {
			parts = append(parts, lineStr)
		}
	}

	if len(parts) == 0 {
		fmt.Println("Claude Statusline")
		return
	}

	fmt.Println(strings.Join(parts, " | "))
}
