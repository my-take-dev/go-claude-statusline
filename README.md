# Claude Statusline

Claude Code用のシンプルなstatuslineツール。Go製で外部依存なし。

## 機能

- **5時間リミット表示** - 使用率と残り時間
- **週間リミット表示** - 7日間の使用率
- **Opusリミット表示** - Opus使用時のみ表示
- **コンテキスト使用量** - 現在のコンテキストウィンドウ使用率
- **5分キャッシュ** - APIへのリクエストを最小限に

## 表示例

```
Opus 4.5 | 5h: 23% (2h45m) | 7d: 45% | Ctx: 12%
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

コマンドラインから直接実行：

```bash
# 単体テスト
./claude-statusline.exe

# Claude Codeの入力をシミュレート
echo '{"model":{"display_name":"Claude Opus 4.5"},"context_window":{"used_percentage":0.12}}' | ./claude-statusline.exe
```

## 要件

- Claude Codeにログイン済み（`claude --login`）
- `~/.claude/.credentials.json` が存在すること

## キャッシュ

Usage APIのレスポンスは5分間キャッシュされます。キャッシュファイル：

- Windows: `%USERPROFILE%\.claude\.statusline-cache.json`
- macOS/Linux: `~/.claude/.statusline-cache.json`

## トラブルシューティング

### トークンが見つからない

```bash
claude --login
```

でログインしてください。

### APIエラー

ネットワーク接続を確認してください。エラー時は古いキャッシュがあればそれを使用します。

## ライセンス

MIT
