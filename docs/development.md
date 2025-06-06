# 開発ガイド

[← トップページに戻る](./index.md)

## 🔧 開発コマンド詳細

### Makefile コマンド
| コマンド | 説明 | 使用場面 |
|---------|------|----------|
| `make build` | アプリケーションをビルド | リリース前の最終確認 |
| `make run` | アプリケーションを起動 | 通常の動作確認 |
| `make watch` | Air使用でホットリロード起動 | **開発時推奨** |
| `make docker-run` | PostgreSQL + Swagger UI + pgweb を起動 | 開発環境セットアップ |
| `make docker-down` | Dockerコンテナを停止・削除 | 開発終了時 |
| `make clean` | ビルド成果物を削除 | クリーンビルド時 |
| `make lint` | コードリンティング | 品質チェック |
| `make format` | コードフォーマット | コード整形 |

### 詳細な使用方法

#### `make watch` - ライブリロード開発

```bash
# ライブリロード開発を開始
make watch

# 初回実行時は air が自動インストールされます
# ファイル変更を検知して自動的にアプリケーションが再起動されます
```

#### `make docker-run` - 開発環境起動

```bash
# データベース環境を起動
make docker-run

# 以下のサービスが起動します:
# - PostgreSQL (localhost:5432)
# - Swagger UI (localhost:8081)
# - pgweb (localhost:8082)
```
#### `make docker-down` - 開発環境停止

```bash
# 起動中のDockerコンテナを停止・削除
make docker-down
```

#### `make lint` - コード品質チェック

```bash
# コードの静的解析を実行
# コードスタイルや潜在的なバグを検出します
make lint
# コードスタイルや潜在的なバグを検出します
```

#### `make format` - コード整形

```bash
# コードのフォーマットを実行
# gofmtを使用してコードを整形します
make format
```
