# taskeru - 実装仕様書

## プロジェクト概要
Goで実装されたCLIタスク管理ツール。todo.txtよりもリッチな表現が可能で、各タスクに詳細なMarkdownノートを追加できる。

## アーキテクチャ

### ディレクトリ構造
```
taskeru/
├── main.go           # エントリーポイント
├── cmd/
│   ├── add.go        # addコマンド
│   ├── list.go       # ls/listコマンド  
│   ├── edit.go       # edit/eコマンド
│   └── root.go       # ルートコマンドとCLI設定
├── internal/
│   ├── task.go       # Task構造体と基本操作
│   ├── storage.go    # ファイルI/O（atomic write）
│   └── ui.go         # Bubble Tea UI
├── go.mod
├── go.sum
└── README.md
```

## データ構造

### Task構造体
```go
type Task struct {
    ID       string    `json:"id"`
    Title    string    `json:"title"`
    Created  time.Time `json:"created"`
    DueDate  *time.Time `json:"due_date,omitempty"`
    Priority string    `json:"priority,omitempty"` // "high", "medium", "low"
    Status   string    `json:"status"`             // "todo", "in_progress", "done"
    Note     string    `json:"note,omitempty"`     // Markdown形式
}
```

### ストレージ仕様
- データファイル: `~/todo.json` (環境変数`TASKERU_FILE`で変更可能)
- 形式: JSONL（JSON Lines）形式 - 1行1タスク
- エディタでの編集がしやすいよう、各タスクは独立した行に保存

例:
```jsonl
{"id":"1","title":"プレゼン準備","created":"2025-08-19T10:00:00Z","status":"todo"}
{"id":"2","title":"買い物","created":"2025-08-19T11:00:00Z","due_date":"2025-08-20T18:00:00Z","priority":"high","status":"todo","note":"# 買い物リスト\n\n- 牛乳\n- パン\n- 卵"}
```

## 依存関係

```go
// go.mod
module taskeru

go 1.21

require (
    github.com/charmbracelet/bubbletea v0.24.2
    github.com/google/uuid v1.3.0
)
```

## 実装詳細

### 1. CLIコマンド構造
```go
// main.go
func main() {
    if len(os.Args) == 1 {
        // デフォルトでls実行
        listTasks()
        return
    }
    
    switch os.Args[1] {
    case "add":
        addTask(os.Args[2:])
    case "ls", "list":
        listTasks()
    case "edit", "e":
        editTask(os.Args[2:])
    }
}
```

### 2. Atomic Write実装
```go
// storage.go
func SaveTasks(tasks []Task, filepath string) error {
    // 一時ファイルに書き込み → rename でatomic write
    // データ整合性を保証
}
```

### 3. Bubble Tea UI
```go
// ui.go  
type TaskSelector struct {
    tasks    []Task
    cursor   int
    selected int
}

// キーバインド:
// - j/k: カーソル移動
// - Enter: 選択
// - q/Esc: キャンセル
```

### 4. Markdownエディタ統合
```go
// edit.go
func editTaskNote(task *Task) error {
    // 1. 一時ファイル作成: # ${TITLE}\n\n${NOTE}
    // 2. $EDITORで編集（デフォルト: vim）
    // 3. ファイル解析してタスクに反映
    // 4. atomic writeで保存
}
```

## 編集フロー

1. `taskeru edit` 実行
2. Bubble Tea UIでタスク一覧表示
3. j/kで選択、Enterで決定
4. 選択されたタスクを一時ファイルに以下の形式で出力:
   ```markdown
   # タスクタイトル
   
   既存のノート内容（あれば）
   ```
5. エディタ（vim）で編集
6. 編集後、ファイルを解析してタスクのtitleとnoteを更新
7. `~/todo.json`にatomic writeで保存

## エラーハンドリング

- ファイルが存在しない場合は空のタスクリストとして扱う
- 不正なJSONL行はスキップして警告表示
- エディタが異常終了した場合は変更を破棄
- atomic writeでデータ整合性を保証

## 環境変数

- `TASKERU_FILE`: データファイルのパス（デフォルト: `~/todo.json`）
- `EDITOR`: 使用するエディタ（デフォルト: `vim`）

## コマンド仕様

### add コマンド
- タスクを追加
- UUIDで一意のIDを生成
- デフォルトステータスは"todo"

### ls/list コマンド
- タスク一覧を表示
- 引数なしのデフォルト動作

### edit/e コマンド  
- インタラクティブUIでタスク選択
- Markdownエディタで編集

## 今後の拡張可能性

- フィルタリング機能（priority, status別）
- 期限切れタスクの警告
- 完了タスクのアーカイブ
- カラー表示
- 統計情報表示

## テスト方針

- 単体テスト: 各パッケージの機能をテスト
- 統合テスト: CLIコマンドの動作確認
- atomic writeのテスト: データ整合性の検証

## ビルドとデプロイ

```bash
# ビルド
go build -o taskeru

# テスト実行
go test ./...

# リント
golangci-lint run
```

## テスト実行時の注意事項

**重要**: テストやデバッグを実行する際は、必ず環境変数 `TASKERU_FILE` を設定して、実際のユーザーデータ (`~/todo.json`) を使用しないようにしてください。

```bash
# テスト用の一時ファイルを使用
export TASKERU_FILE=/tmp/test-todo.json
./taskeru add "テストタスク"
./taskeru ls

# またはコマンドごとに指定
TASKERU_FILE=/tmp/test-todo.json ./taskeru add "テストタスク"
TASKERU_FILE=/tmp/test-todo.json ./taskeru ls

# テスト後はファイルを削除
rm /tmp/test-todo.json
```

これにより、実際のタスクデータを誤って変更・削除することを防げます。