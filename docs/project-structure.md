# プロジェクト構造

## 🌳 ディレクトリ構成

```
tsumitan-backend/
├── cmd/api/main.go             # アプリケーション起動
├── internal/                   # 内部パッケージ
│   ├── auth/                   # Firebase JWT認証
│   ├── database/               # PostgreSQL接続管理
│   ├── models/                 # データモデル
│   └── server/                 # Webサーバー
│       ├── server.go           # サーバー設定
│       └── routes.go           # API ルーティング
├── docs/                       # ドキュメント
├── docker-compose.yml          # 開発環境
├── openapi.yml                 # API仕様書
├── Makefile                    # ビルド・実行コマンド
├── go.mod                      # Go依存関係
└── .env.example                # 環境変数テンプレート
```

## 📂 主要コンポーネント

### `cmd/api/main.go`
- アプリケーション起動
- グレースフルシャットダウン

### `internal/auth/`
- Firebase JWT認証
- Google公開鍵キャッシュ
- Echo認証ミドルウェア


### `internal/server/`
- **server.go**: サーバー設定
- **routes.go**: APIルーティングとハンドラー

## 🔧 設定ファイル

### `docker-compose.yml`
- PostgreSQL (psql_bp)
- Swagger UI (swagger-ui)
- pgweb 管理UI

### `Makefile`
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

### 主要依存関係
- Echo v4 - Webフレームワーク
- GORM - ORM
- JWT v5 - JWT処理
- godotenv - 環境変数
