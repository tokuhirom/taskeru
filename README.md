# taskeru

Goで実装されたシンプルかつ強力なCLIタスク管理ツール

## 概要

`taskeru`は個人のタスク管理のためのCLIツールです。todo.txtよりもリッチな表現が可能で、各タスクに詳細なMarkdownノートを追加できます。

## 機能

### 基本機能
- タスクの追加、一覧表示、編集、削除
- 各タスクに対するMarkdown形式の詳細ノート
- プロジェクトタグ（`+project`形式）によるタスク分類
- 優先度管理（A-Z）
- ステータス管理（TODO/DOING/WAITING/DONE/WONTDO）

### UI機能
- インタラクティブなタスク選択（Bubble Tea UI）
- Kanbanボード表示
- プロジェクトビュー
- 日本語完全対応

### データ管理
- JSONLファイル形式でのデータ保存
- 自動タイムスタンプ機能（設定可能）
- ゴミ箱機能（削除タスクの自動保存）

## インストール

```bash
go build -o taskeru
```

## 使用方法

### コマンドライン

#### タスクの追加
```bash
taskeru add "プレゼン準備"
taskeru add "買い物に行く +shopping +urgent"
```

#### タスク一覧表示
```bash
taskeru        # インタラクティブモード（デフォルト）
taskeru ls     # シンプルなリスト表示
```

#### タスクの編集
```bash
taskeru edit   # インタラクティブ選択してエディタで編集
taskeru e      # 短縮形
```

#### Kanbanボード表示
```bash
taskeru kanban # Kanbanビューを表示
```

### インタラクティブモード

引数なしで `taskeru` を実行すると、インタラクティブモードが起動します。

#### キーバインド（リストビュー）
- `j`/`k` または `↑`/`↓`: カーソル移動
- `space`: タスクの完了/未完了切り替え
- `s`: ステータス変更（TODO→DOING→WAITING→DONE→WONTDO）
- `+`/`-`: 優先度の上げ下げ
- `c`: 新規タスク作成
- `e`: タスク編集（Vimが開く）
- `d`: タスク削除（確認あり）
- `p`: プロジェクトビュー表示
- `a`: 全タスク表示（古い完了タスクも含む）
- `r`: リロード
- `g`/`G`: 先頭/末尾へジャンプ
- `ctrl+u`/`ctrl+d`: ページアップ/ダウン
- `q`: 終了

### エディタでの編集

タスクを編集すると、以下の形式でVimが開きます：

```markdown
# タスクタイトル +project1 +project2

## 2025-08-21(Thu) 14:30  ← タイムスタンプ（設定で有効時）

ここにMarkdown形式でメモを記入...
```

プロジェクトタグはタイトル行で編集できます。

## 設定

### 設定ファイル

`~/.config/taskeru/config.toml` に設定ファイルを配置できます。

```bash
# 設定ファイルの初期化
taskeru init-config
```

#### 設定項目

```toml
[editor]
# タスク編集時に自動的にタイムスタンプを追加
add_timestamp = false  # true で有効化
```

### 環境変数
- `EDITOR`: 使用するエディタ（デフォルト: `vim`）

### コマンドラインオプション
- `-t <file>`: タスクファイルのパスを指定（環境変数より優先）

## データ形式

タスクはJSONL（JSON Lines）形式で保存されます：

```json
{"id":"uuid","title":"タスク名","status":"TODO","priority":"A","projects":["work","urgent"]}
```

削除されたタスクは `~/todo.trash.json` に自動的にバックアップされます。

## ライセンス

MIT License
