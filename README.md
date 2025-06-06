# 積み単 ( Tsumitan )

<!-- デプロイしたらそれについても書く。デプロイ先のURLやサービスなど-->

言語学習を支援するGoベースのWebAPIアプリケーションのバックエンドです。
フロントエンドは[こちら](https://github.com/geek-hackathon-vol6-team20/tsumitan-frontend)

## 📚 プロジェクト構成

このプロジェクトは以下の技術スタックを使用しています：

### Backend Framework
- **Go 1.24.2** - メインプログラミング言語
- **Echo v4** - 高性能WebフレームワーでAPIサーバーを構築

### Database & ORM
- **PostgreSQL** - メインデータベース
- **GORM** - Go用のORM（Object-Relational Mapping）ライブラリ
- **pgweb** - PostgreSQLのWebベース管理ツール

### Authentication
- **Firebase Authentication** - 認証システム
- **JWT v5** - JSON Web Token処理ライブラリ

### Development Tools
- **Air** - ホットリロード機能付きの開発サーバー
- **Docker & Docker Compose** - コンテナ化による開発環境の構築
- **Make** - ビルドタスクの自動化

### API Documentation
- **OpenAPI 3.0** - API仕様書の標準規格
- **Swagger UI** - インタラクティブなAPI文書の表示

### Code Quality
- **golangci-lint** - 静的解析ツール
- **gofmt** - コードフォーマッター

## 🏗️ アーキテクチャ

```
├── cmd/api/          # アプリケーションエントリーポイント
├── internal/
│   ├── auth/         # Firebase JWT認証
│   ├── database/     # データベース接続・操作
│   ├── models/       # データモデル (Word等)
│   └── server/       # HTTPサーバー・ルーティング
└── docs/            # プロジェクトドキュメント
```

## 🚀 Getting Started

### 前提条件

- Go 1.21以上
- Docker & Docker Compose
- Make

### 環境設定

1. 環境変数ファイルを設定：
```bash
cp .env.example .env
# .envファイルを編集してFirebaseやデータベースの設定を行う
```

2. データベースコンテナを起動：
```bash
make docker-run
```

3. ライブリロードのついたサーバーを起動：
```bash
make watch
```

## ドキュメント
ドキュメントは[こちら](docs/index.md)