# リセット時間表示の修正

## Context

Claude Code の `resets_at` は秒単位の Unix タイムスタンプ（例: `1773997200`）だが、コードは `time.UnixMilli()` で解釈しているため 1970年1月の過去日時となり、`remaining <= 0` → 常に `"now"` と表示されてしまう。また、7日間のリセット表示が `1d1h` のように分を省略している。

## 変更対象

- `main.go` の `formatResetTime()` 関数（210-232行目）

## 変更内容

### 1. タイムスタンプの単位を秒に修正

```go
// 変更前
resetsAt := time.UnixMilli(resetsAtMs)

// 変更後
resetsAt := time.Unix(resetsAt, 0)
```

パラメータ名も `resetsAtMs` → `resetsAt` にリネーム（ミリ秒ではないため）。

### 2. 日数表示に分を追加

```go
// 変更前
if days > 0 {
    return fmt.Sprintf("%dd%dh", days, hours)
}

// 変更後
if days > 0 {
    return fmt.Sprintf("%dd%dh%dm", days, hours, minutes)
}
```

### 3. 呼び出し元のパラメータ名更新

`fmtBarReset()` 関数のパラメータ名も `resetsAtMs` → `resetsAt` に合わせてリネーム。コメントの「ミリ秒」も「秒」に修正。

## 変更後の表示例

| ケース | 変更前 | 変更後 |
|--------|--------|--------|
| 5h リセットまで1時間45分 | `now` | `1h45m` |
| 7d リセットまで1日1時間45分 | `now` | `1d1h45m` |
| リセット済み | `now` | `now` |
| 1分未満 | `now` | `<1m` |

## 検証方法

```bash
echo '{"model":{"display_name":"Claude Sonnet 4.6"},"context_window":{"used_percentage":12},"rate_limits":{"five_hour":{"used_percentage":22,"resets_at":1773997200},"seven_day":{"used_percentage":68,"resets_at":1774062000}}}' | ./claude-statusline.exe
```

リセット時間が `now` ではなく適切な残り時間（例: `1h45m`, `1d1h45m`）で表示されることを確認。
