# データベース

PostgreSQL + GORM、管理UI: http://localhost:8082

## 📊 データモデル

---

## 📊 データモデル

### Word モデル

ユーザーの単語学習データを管理するメインテーブルです。

```go
type Word struct {
    UserID       string    `gorm:"primaryKey" json:"user_id"`
    Word         string    `gorm:"primaryKey" json:"word"`
    SearchCount  int       `json:"search_count"`
    ReviewCount  int       `json:"review_count"`
    LastReviewed time.Time `json:"last_reviewed"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

#### フィールド詳細
| フィールド | 型 | 制約 | 説明 |
|-----------|-----|------|------|
| `UserID` | string | PRIMARY KEY | Firebase UID |
| `Word` | string | PRIMARY KEY | 検索した英単語 |
| `SearchCount` | int | NOT NULL | 検索回数 |
| `ReviewCount` | int | NOT NULL | 復習回数 |
| `LastReviewed` | time.Time | - | 最後の復習日時 |
| `CreatedAt` | time.Time | AUTO | 初回検索日時 |
| `UpdatedAt` | time.Time | AUTO | 最終更新日時 |

#### 複合主キー
- `UserID` + `Word` の組み合わせでユニーク
- 同じユーザーが同じ単語を複数回検索した場合、`SearchCount`が増加
