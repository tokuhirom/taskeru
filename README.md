# taskeru

Goで実装されたシンプルなCLIタスク管理ツール

## 概要

`taskeru`は個人のタスク管理のためのCLIツールです。todo.txtよりもリッチな表現が可能で、各タスクに詳細なMarkdownノートを追加できます。

## 機能

- タスクの追加、一覧表示、編集
- 各タスクに対するMarkdown形式の詳細ノート
- インタラクティブなタスク選択（Bubble Tea UI）
- Vimエディタとの統合
- JSONLファイル形式でのデータ保存

## インストール

```bash
go build -o taskeru
```

## 使用方法

### タスクの追加
```bash
taskeru add "プレゼン準備"
taskeru add "買い物に行く"
```

### タスク一覧表示
```bash
taskeru        # デフォルトコマンド（ls と同じ）
taskeru ls
```

### タスクの編集
```bash
taskeru edit   # インタラクティブ選択
taskeru e      # 短縮形
```

## データ構造

### タスクの構造
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

### ファイル形式
- データファイル: `~/todo.json` (環境変数`TASKERU_FILE`で変更可能)
- 形式: 1行1タスクのJSONL（JSON Lines）形式
- エディタでの編集がしやすいよう、各タスクは独立した行に保存

例:
```jsonl
{"id":"1","title":"プレゼン準備","created":"2025-08-19T10:00:00Z","status":"todo"}
{"id":"2","title":"買い物","created":"2025-08-19T11:00:00Z","due_date":"2025-08-20T18:00:00Z","priority":"high","status":"todo","note":"# 買い物リスト\n\n- 牛乳\n- パン\n- 卵"}
```

## 実装仕様

### 必要な依存関係
```go
// go.mod
module taskeru

go 1.21

require (
    github.com/charmbracelet/bubbletea v0.24.2
    github.com/google/uuid v1.3.0
)
```

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

### 主要な実装ポイント

#### 1. Atomic Write
```go
// storage.go
func SaveTasks(tasks []Task, filepath string) error {
    // 一時ファイルに書き込み → rename でatomic write
}
```

#### 2. Bubble Tea UI
```go
// ui.go  
type TaskSelector struct {
    tasks    []Task
    cursor   int
    selected int
}

// j/kキーでナビゲーション
// Enterで選択
```

#### 3. Markdownエディタ統合
```go
// edit.go
func editTaskNote(task *Task) error {
    // 一時ファイル作成: # ${TITLE}\n\n${NOTE}
    // vimで編集
    // ファイル解析してタスクに反映
}
```

#### 4. CLIコマンド構造
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

### 設定

#### 環境変数
- `TASKERU_FILE`: データファイルのパス（デフォルト: `~/todo.json`）
- `EDITOR`: 使用するエディタ（デフォルト: `vim`）

### 編集フロー

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

### エラーハンドリング

- ファイルが存在しない場合は空のタスクリストとして扱う
- 不正なJSONL行はスキップして警告表示
- エディタが異常終了した場合は変更を破棄

### 今後の拡張可能性

- フィルタリング機能（priority, status別）
- 期限切れタスクの警告
- 完了タスクのアーカイブ
- カラー表示
- 統計情報表示

## ライセンス

MIT License
