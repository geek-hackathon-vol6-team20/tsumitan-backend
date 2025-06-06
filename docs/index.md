# 英単語学習アプリ バックエンドAPI

## 📖 プロジェクト概要

このプロジェクトは、英単語学習アプリのバックエンドAPIです。ユーザーの単語検索履歴、復習セッション、単語のメタ情報を管理し、学習効果を最大化するための機能を提供します。

### 主な機能
- 🔍 **単語検索履歴の記録**: ユーザーが検索した単語の履歴を追跡
- 📚 **復習セッション管理**: 効果的な復習スケジュールをサポート
- 📊 **学習統計**: 検索回数・復習回数などの学習データを管理
- 🔐 **Firebase認証**: セキュアなユーザー認証システム

---

## 🛠 技術スタック

| カテゴリ | 技術 | バージョン |
|---------|------|-----------|
| **言語** | Go | 1.24.2 |
| **Webフレームワーク** | Echo | v4.13.4 |
| **データベース** | PostgreSQL | latest |
| **ORM** | GORM | v1.30.0 |
| **認証** | Firebase | - |
| **コンテナ化** | Docker & Docker Compose | - |
| **ドキュメント** | Swagger/OpenAPI | 3.0.0 |

---

## 📚 ドキュメント

### 📖 開発者向けガイド
- **[プロジェクト構造](./project-structure.md)** - ファイル・ディレクトリ構成の詳細
- **[開発ガイド](./development.md)** - 開発時のコマンドとワークフロー

### 🔧 技術仕様
- **[API仕様](./api.md)** - REST API エンドポイントとSwagger
- **[認証システム](./authentication.md)** - Firebase JWT認証の詳細
- **[データベース](./database.md)** - スキーマとデータモデル


## 🚀 必要な環境

- **Go 1.24.2以上**
- **Docker & Docker Compose**
- **make**

## 🔧 クイックスタート

```bash
# 1. リポジトリクローン
git clone <repository-url>
cd tsumitan-backend

# 2. 環境変数設定
cp .env.example .env
# .envを編集してDBパスワード等を設定

# 4. データベース起動
make docker-run

# 5. アプリケーション起動
make watch
```

## ✅ 動作確認

起動後、以下で正常動作を確認できます：

- **API**: http://localhost:8080/
- **Swagger UI**: http://localhost:8081
- **DB管理（pgweb）**: http://localhost:8082

## 🛠️ 利用可能なコマンド

プロジェクト特有のMakefileコマンド：

```bash
make build        # アプリケーションをビルド
make run          # アプリケーション起動
make docker-run   # PostgreSQL + Swagger UI + pgweb を起動
make docker-down  # Dockerコンテナを停止・削除
make clean        # ビルド成果物を削除
make watch        # Air使用でホットリロード起動
make lint         # コードリンティング
make format       # コードフォーマット
```
