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

編集モードでは:
- `j`/`k` キーで上下移動
- `Enter` で選択
- `q` または `Esc` でキャンセル

## 設定

### 環境変数
- `TASKERU_FILE`: データファイルのパス（デフォルト: `~/todo.json`）
- `EDITOR`: 使用するエディタ（デフォルト: `vim`）

## ライセンス

MIT License