# Claude Statusline

Claude Code用のシンプルなstatuslineツール。Go製で外部依存なし。

## 機能

- **モデル名表示** - 使用中のモデル名を短縮表示
- **コンテキスト使用量** - コンテキストウィンドウの使用率
- **セッションコスト** - セッション累計コスト（USD）
- **セッション時間** - セッション経過時間
- **行数変更** - 追加/削除行数
- **後方互換性** - 新旧どちらのClaude Code JSONフォーマットにも対応

## 表示例

```
Opus 4.6 | Ctx: 12% | $1.23 | 45m | +150/-30
```

旧フォーマット（コスト情報なし）：
```
Opus 4.5 | Ctx: 12%
```

## インストール

### ビルド

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o claude-statusline.exe

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o claude-statusline

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o claude-statusline

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o claude-statusline
```

### 配置

ビルドした実行ファイルを配置します：

**Windows:**
```
%USERPROFILE%\.claude\claude-statusline.exe
```

**macOS/Linux:**
```
~/.claude/claude-statusline
```

### 設定

`~/.claude/settings.json` に追加：

**Windows:**
```json
{
  "statusLine": {
    "type": "command",
    "command": "%USERPROFILE%\\.claude\\claude-statusline.exe"
  }
}
```

**macOS/Linux:**
```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/claude-statusline"
  }
}
```

## 動作確認

```bash
# 新フォーマット（全フィールド）
echo '{"model":{"display_name":"Claude Opus 4.6"},"context_window":{"used_percentage":0.12},"cost":{"total_cost_usd":1.23,"total_duration_ms":2700000,"total_lines_added":150,"total_lines_removed":30}}' | ./claude-statusline

# 旧フォーマット（後方互換性テスト）
echo '{"model":{"display_name":"Claude Opus 4.5"},"context_window":{"used_percentage":0.12}}' | ./claude-statusline
```

## 仕組み

Claude Codeがstdinで送信するJSONデータを解析し、ステータスラインを構築します。

### 入力JSON

Claude Codeは以下のようなJSONをstdinにパイプします：

```json
{
  "model": {
    "id": "claude-opus-4-6",
    "display_name": "Claude Opus 4.6"
  },
  "context_window": {
    "used_percentage": 0.12
  },
  "cost": {
    "total_cost_usd": 1.23,
    "total_duration_ms": 2700000,
    "total_lines_added": 150,
    "total_lines_removed": 30
  }
}
```

### 表示セグメント

| セグメント | 例 | 条件 |
|-----------|-----|------|
| モデル名 | `Opus 4.6` | display_nameが存在 |
| コンテキスト | `Ctx: 12%` | 使用率 > 0% |
| コスト | `$1.23` | コスト > $0 |
| セッション時間 | `45m` | 時間 > 0 |
| 行数変更 | `+150/-30` | 変更がある場合 |

## ライセンス

MIT
