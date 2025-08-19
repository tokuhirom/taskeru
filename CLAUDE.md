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
    ID          string     `json:"id"`
    Title       string     `json:"title"`
    Created     time.Time  `json:"created"`
    Updated     time.Time  `json:"updated"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
    DueDate     *time.Time `json:"due_date,omitempty"`
    Priority    string     `json:"priority,omitempty"` // "high", "medium", "low"
    Status      string     `json:"status"`             // "todo", "in_progress", "done"
    Note        string     `json:"note,omitempty"`     // Markdown形式
}
```

### ストレージ仕様
- データファイル: `~/todo.json` (`-t` オプションまたは環境変数`TASKERU_FILE`で変更可能)
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

### グローバルオプション
- `-t <file>`: タスクファイルのパスを指定（TASKERU_FILE環境変数より優先）

### add コマンド
- タスクを追加
- UUIDで一意のIDを生成
- デフォルトステータスは"todo"
- `+project` 形式でプロジェクトタグをサポート
- 例: `taskeru add "タスク名 +work +urgent"`

### ls/list コマンド
- タスク一覧を表示
- プロジェクトはシアン色で表示
- 古い完了タスクは自動的に非表示

### edit/e コマンド  
- インタラクティブUIでタスク選択
- Markdownエディタで編集

### インタラクティブモード（引数なし）
- `j/k` または `↑/↓`: カーソル移動
- `space`: タスクの完了/未完了切り替え
- `a`: 全タスク表示（古い完了タスクも含む）
- `c`: 新規タスク作成（プロジェクトタグ対応）
- `e`: 選択したタスクを編集
- `d`: タスク削除（確認あり）
- `p`: プロジェクトビュー表示
- `r`: タスク一覧を再読み込み
- `q`: 終了

## 今後の拡張可能性

- フィルタリング機能（priority, status別）
- 期限切れタスクの警告
- 完了タスクのアーカイブ
- カラー表示
- 統計情報表示

## todo.txt仕様との機能比較と今後の実装予定

現在のtaskeruは独自のJSON形式を採用していますが、todo.txt形式の標準的な機能のうち、以下の機能が未実装です：

### 1. プロジェクト機能 (+Project)
- **todo.txt仕様**: `+` で始まるプロジェクトタグを複数設定可能
- **実装例**: `(A) レポート作成 +仕事 +Q4目標`
- **利点**: タスクをプロジェクト単位で分類・フィルタリング可能
- **実装案**: Task構造体に `Projects []string` フィールドを追加

### 2. コンテキスト機能 (@Context)
- **todo.txt仕様**: `@` で始まるコンテキストタグを複数設定可能
- **実装例**: `買い物に行く @外出 @週末`
- **利点**: 場所や状況に応じたタスクの分類
- **実装案**: Task構造体に `Contexts []string` フィールドを追加

### 3. ~~完了日の専用記録~~ ✅ 実装済み
- **todo.txt仕様**: 完了時に専用の完了日フィールド
- **実装例**: `x 2024-01-15 2024-01-10 タスク名` (完了日 作成日 タスク)
- **現状**: `CompletedAt *time.Time` フィールドで実装済み
- **備考**: `SetStatus("done")`時に自動的に完了時刻が記録される

### 4. 優先度の詳細化
- **todo.txt仕様**: (A)〜(Z)の26段階の優先度
- **現状**: high/medium/lowの3段階
- **実装案**: Priority を `string` から `rune` (A-Z) に変更、または両方サポート

### 5. カスタムメタデータ (key:value)
- **todo.txt仕様**: 任意のkey:valueペアを設定可能
- **実装例**: `due:2024-01-01 recur:weekly estimate:2h`
- **利点**: 柔軟な情報追加が可能
- **実装案**: Task構造体に `Metadata map[string]string` フィールドを追加

### 6. 自動ソート機能
- **todo.txt仕様**: 優先度順 → 未完了 → 完了の順で自動ソート
- **現状**: 作成順のまま表示
- **実装案**: インタラクティブモードにソートオプションを追加

### 7. 繰り返しタスク
- **todo.txt拡張仕様**: `recur:` タグで繰り返し設定
- **実装例**: `recur:daily`, `recur:weekly`, `recur:monthly`
- **利点**: 定期的なタスクの自動生成
- **実装案**: メタデータとして実装し、完了時に次のタスクを自動生成

### 8. タスク間の依存関係
- **todo.txt拡張仕様**: `depends:` タグで依存関係を設定
- **実装例**: `depends:task-id-123`
- **利点**: タスクの順序関係を明確化
- **実装案**: メタデータとして実装し、UIで依存関係を可視化

### 9. タグによるフィルタリング
- **機能**: プロジェクトやコンテキストでフィルタリング
- **実装例**: `taskeru ls +仕事` でプロジェクト「仕事」のタスクのみ表示
- **実装案**: listコマンドにフィルタオプションを追加

これらの機能を段階的に実装することで、taskeruをより強力で柔軟なタスク管理ツールに進化させることができます。

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

**重要**: テストやデバッグを実行する際は、実際のユーザーデータ (`~/todo.json`) を使用しないよう、以下のいずれかの方法でファイルパスを指定してください。

### 1. -t オプションを使用（推奨）

```bash
# -t オプションで一時ファイルを指定
./taskeru -t /tmp/test-todo.json add "テストタスク"
./taskeru -t /tmp/test-todo.json ls
./taskeru -t /tmp/test-todo.json  # インタラクティブモード

# テスト後はファイルを削除
rm /tmp/test-todo.json
```

### 2. 環境変数を使用

```bash
# 環境変数で指定
export TASKERU_FILE=/tmp/test-todo.json
./taskeru add "テストタスク"
./taskeru ls

# またはコマンドごとに指定
TASKERU_FILE=/tmp/test-todo.json ./taskeru add "テストタスク"
TASKERU_FILE=/tmp/test-todo.json ./taskeru ls
```

### オプションの優先順位

1. `-t` オプション（最優先）
2. `TASKERU_FILE` 環境変数
3. デフォルト `~/todo.json`

これにより、実際のタスクデータを誤って変更・削除することを防げます。
