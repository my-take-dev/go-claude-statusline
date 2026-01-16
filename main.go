package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// === 設定 ===
const (
	CacheExpiry = 5 * time.Minute // キャッシュ有効期限
	APIEndpoint = "https://api.anthropic.com/api/oauth/usage"
	UserAgent   = "claude-code/2.0.31"
	BetaHeader  = "oauth-2025-04-20"
)

// === データ構造 ===

// Claude Codeからstdinで渡されるJSON
type ClaudeCodeInput struct {
	Model struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow struct {
		ContextWindowSize int     `json:"context_window_size"`
		UsedPercentage    float64 `json:"used_percentage"`
		CurrentUsage      *struct {
			InputTokens            int `json:"input_tokens"`
			CacheCreationTokens    int `json:"cache_creation_input_tokens"`
			CacheReadTokens        int `json:"cache_read_input_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`
}

// credentials.jsonの構造
type Credentials struct {
	ClaudeAiOauth *struct {
		AccessToken string `json:"accessToken"`
	} `json:"claudeAiOauth"`
}

// Usage APIのレスポンス
type UsageResponse struct {
	FiveHour *UsageLimit `json:"five_hour"`
	SevenDay *UsageLimit `json:"seven_day"`
	SevenDayOpus *UsageLimit `json:"seven_day_opus"`
}

type UsageLimit struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

// キャッシュファイルの構造
type CachedUsage struct {
	FetchedAt    time.Time     `json:"fetched_at"`
	Usage        *UsageResponse `json:"usage"`
}

// === ヘルパー関数 ===

// Claudeの設定ディレクトリを取得
func getClaudeDir() string {
	// 環境変数で上書き可能
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}

// credentials.jsonからOAuthトークンを取得
func getOAuthToken() (string, error) {
	claudeDir := getClaudeDir()
	if claudeDir == "" {
		return "", fmt.Errorf("cannot find claude directory: set CLAUDE_CONFIG_DIR or ensure home directory is accessible")
	}
	
	credPath := filepath.Join(claudeDir, ".credentials.json")
	data, err := os.ReadFile(credPath)
	if err != nil {
		return "", fmt.Errorf("cannot read credentials: %w", err)
	}
	
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", fmt.Errorf("cannot parse credentials: %w", err)
	}
	
	if creds.ClaudeAiOauth == nil || creds.ClaudeAiOauth.AccessToken == "" {
		return "", fmt.Errorf("no OAuth token found")
	}
	
	return creds.ClaudeAiOauth.AccessToken, nil
}

// キャッシュファイルのパス
func getCachePath() string {
	claudeDir := getClaudeDir()
	if claudeDir == "" {
		// フォールバック: 一時ディレクトリ
		return filepath.Join(os.TempDir(), "claude-statusline-cache.json")
	}
	return filepath.Join(claudeDir, ".statusline-cache.json")
}

// キャッシュから読み込み
func loadCache() (*CachedUsage, error) {
	data, err := os.ReadFile(getCachePath())
	if err != nil {
		return nil, err
	}
	
	var cached CachedUsage
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}
	
	return &cached, nil
}

// キャッシュに保存（アトミック書き込み）
func saveCache(usage *UsageResponse) error {
	cached := CachedUsage{
		FetchedAt: time.Now(),
		Usage:     usage,
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	cachePath := getCachePath()

	// 一時ファイルに書き込み（同じディレクトリで作成してリネームを確実に）
	dir := filepath.Dir(cachePath)
	tempFile, err := os.CreateTemp(dir, ".statusline-cache-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// 失敗時のクリーンアップ
	defer func() {
		if tempPath != "" {
			os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// パーミッションを設定
	if err := os.Chmod(tempPath, 0600); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// アトミックにリネーム
	if err := os.Rename(tempPath, cachePath); err != nil {
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	tempPath = "" // リネーム成功、クリーンアップ不要
	return nil
}

// Usage APIを叩く
func fetchUsage(token string) (*UsageResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", APIEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", BetaHeader)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call usage API: %w", err)
	}
	defer resp.Body.Close()

	// レスポンスボディのサイズ制限（1MB）
	limitedReader := io.LimitReader(resp.Body, 1<<20)

	if resp.StatusCode != 200 {
		// レスポンスボディを読み捨て（機密情報が含まれる可能性があるため表示しない）
		_, _ = io.ReadAll(limitedReader)
		return nil, fmt.Errorf("usage API returned status %d", resp.StatusCode)
	}

	var usage UsageResponse
	if err := json.NewDecoder(limitedReader).Decode(&usage); err != nil {
		return nil, fmt.Errorf("failed to decode usage response: %w", err)
	}

	return &usage, nil
}

// キャッシュ付きでUsageを取得
func getUsage() (*UsageResponse, time.Time, error) {
	// キャッシュをチェック
	cached, err := loadCache()
	if err == nil && time.Since(cached.FetchedAt) < CacheExpiry {
		return cached.Usage, cached.FetchedAt, nil
	}

	// トークン取得
	token, err := getOAuthToken()
	if err != nil {
		// トークンがない場合、古いキャッシュがあれば使う
		if cached != nil {
			cacheAge := time.Since(cached.FetchedAt)
			fmt.Fprintf(os.Stderr, "warning: using cached data from %v ago (token error: %v)\n", cacheAge.Round(time.Minute), err)
			return cached.Usage, cached.FetchedAt, nil
		}
		return nil, time.Time{}, err
	}

	// API呼び出し
	usage, err := fetchUsage(token)
	if err != nil {
		// APIエラーの場合、古いキャッシュがあれば使う
		if cached != nil {
			cacheAge := time.Since(cached.FetchedAt)
			fmt.Fprintf(os.Stderr, "warning: using cached data from %v ago (API error: %v)\n", cacheAge.Round(time.Minute), err)
			return cached.Usage, cached.FetchedAt, nil
		}
		return nil, time.Time{}, err
	}

	// キャッシュ保存
	fetchedAt := time.Now()
	if err := saveCache(usage); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to save cache: %v\n", err)
	}

	return usage, fetchedAt, nil
}

// stdinからClaude Codeの入力を読む
func readClaudeCodeInput() (*ClaudeCodeInput, error) {
	// stdinが空かチェック
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat stdin: %w", err)
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// TTYからの入力（パイプではない）
		return nil, nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}
	if len(data) == 0 {
		// 空データはエラーではない（パイプはあるが空の場合）
		return nil, nil
	}

	var input ClaudeCodeInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse Claude Code input JSON: %w", err)
	}

	return &input, nil
}

// 残り時間をフォーマット
func formatTimeRemaining(resetsAt string) string {
	if resetsAt == "" {
		return ""
	}
	
	resetTime, err := time.Parse(time.RFC3339, resetsAt)
	if err != nil {
		return ""
	}
	
	remaining := time.Until(resetTime)
	if remaining < 0 {
		return "0m"
	}
	
	hours := int(remaining.Hours())
	minutes := int(remaining.Minutes()) % 60
	
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// コンテキスト使用率を計算
func calculateContextPercentage(input *ClaudeCodeInput) float64 {
	if input == nil {
		return 0
	}
	
	// used_percentageがあればそれを使用
	if input.ContextWindow.UsedPercentage > 0 {
		// 1.0より大きければ既にパーセント値
		if input.ContextWindow.UsedPercentage > 1.0 {
			return input.ContextWindow.UsedPercentage
		}
		// 0.0-1.0の範囲なら小数形式
		return input.ContextWindow.UsedPercentage * 100
	}
	
	// current_usageから計算
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
	case "Claude Opus 4.5":
		return "Opus 4.5"
	case "Claude Sonnet 4.5":
		return "Sonnet 4.5"
	case "Claude Sonnet 4":
		return "Sonnet 4"
	case "Claude Haiku 4.5":
		return "Haiku 4.5"
	default:
		if len(name) > 12 {
			return name[:12]
		}
		return name
	}
}

func main() {
	// stdinからClaude Codeの入力を読む
	ccInput, inputErr := readClaudeCodeInput()
	if inputErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to read Claude Code input: %v\n", inputErr)
	}

	// Usage APIからデータ取得
	usage, fetchedAt, usageErr := getUsage()

	// 出力を組み立て
	var parts []string

	// モデル名
	if ccInput != nil && ccInput.Model.DisplayName != "" {
		parts = append(parts, shortenModelName(ccInput.Model.DisplayName))
	}

	// 5時間リミット
	if usage != nil && usage.FiveHour != nil {
		remaining := formatTimeRemaining(usage.FiveHour.ResetsAt)
		if remaining != "" {
			parts = append(parts, fmt.Sprintf("5h: %.0f%% (%s)", usage.FiveHour.Utilization, remaining))
		} else {
			parts = append(parts, fmt.Sprintf("5h: %.0f%%", usage.FiveHour.Utilization))
		}
	}

	// 週間リミット
	if usage != nil && usage.SevenDay != nil {
		parts = append(parts, fmt.Sprintf("7d: %.0f%%", usage.SevenDay.Utilization))
	}

	// Opusリミット（使用中の場合のみ）
	if usage != nil && usage.SevenDayOpus != nil && usage.SevenDayOpus.Utilization > 0 {
		parts = append(parts, fmt.Sprintf("Opus: %.0f%%", usage.SevenDayOpus.Utilization))
	}

	// コンテキスト使用量
	if ccInput != nil {
		ctxPct := calculateContextPercentage(ccInput)
		if ctxPct > 0 {
			parts = append(parts, fmt.Sprintf("Ctx: %.0f%%", ctxPct))
		}
	}

	// 更新時刻
	if !fetchedAt.IsZero() {
		parts = append(parts, fmt.Sprintf("@%s", fetchedAt.Format("15:04")))
	}

	// エラーがあり、何も表示できない場合
	if len(parts) == 0 {
		if usageErr != nil {
			fmt.Println("⚠ " + usageErr.Error())
		} else {
			fmt.Println("Claude Statusline")
		}
		return
	}

	// 結合して出力
	fmt.Println(strings.Join(parts, " | "))
}
