# Claude Statusline

Claude Code用のシンプルなstatuslineツール。Go製で外部依存なし。

Claude Codeがstdin経由でJSONデータを渡してくれるので、APIコールやキャッシュは不要です。

## 機能

- **モデル名** - 現在使用中のモデルを短縮表示
- **コンテキスト使用率** - 現在のコンテキストウィンドウ使用率
- **セッションコスト** - 累積コスト（USD）
- **セッション時間** - 経過時間

## 表示例

```
Sonnet 4.6 | 使用率:12% | Ctx: 12% | $0.42 | 5m
```

## インストール

### ビルド

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o claude-statusline.exe

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o claude-statusline

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o claude-statusline

# Linux
GOOS=linux GOARCH=amd64 go build -o claude-statusline
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

または絶対パスで：
```json
{
  "statusLine": {
    "type": "command",
    "command": "C:\\Users\\YourName\\.claude\\claude-statusline.exe"
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

コマンドラインからClaude Codeの入力をシミュレートして確認：

```bash
echo '{"model":{"display_name":"Claude Sonnet 4.6"},"context_window":{"used_percentage":0.12},"cost":{"total_cost_usd":0.42,"total_duration_ms":300000}}' | ./claude-statusline.exe
```

## デバッグモード

デバッグログを有効にするには、フラグファイルを作成します：

**Windows:**
```
%USERPROFILE%\.claude\statusline-debug-mode
```

**macOS/Linux:**
```
~/.claude/statusline-debug-mode
```

ログは同ディレクトリの `statusline-debug.log` に出力されます。

## 要件

- Go 1.18以上（ビルド時のみ）
- Claude Code（実行時）

## ライセンス

MIT
