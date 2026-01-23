# Calcium モジュールレジストリ仕様 (Draft)

## 概要

モジュールのメタデータを GitHub リポジトリで管理し、申請（PR）ベースで登録する。

## 設計方針

1. **GitHub リポジトリがレジストリ** - サーバー不要、透明性が高い
2. **申請型** - PR でモジュールを登録、レビュー後マージ
3. **メタデータのみ管理** - 実コードは作者のリポジトリに置く
4. **静的ホスティング** - GitHub Pages で配信可能

---

## レジストリ構成

### リポジトリ: `calcium-lang/modules`

作者名ベースの階層構造で、スケーラビリティを確保。

```
calcium-lang/modules/
├── README.md
├── index/
│   ├── all.toml            # 全モジュール名リスト（軽量）
│   ├── A.toml              # A で始まる作者のモジュール一覧
│   ├── B.toml
│   └── ...
├── packages/
│   ├── J/
│   │   └── JO/
│   │       └── JOHNDOE/
│   │           ├── author.toml     # 作者情報
│   │           ├── http/
│   │           │   ├── meta.toml
│   │           │   ├── 1.0.0.toml
│   │           │   └── 1.1.0.toml
│   │           └── json-parser/
│   │               ├── meta.toml
│   │               └── 2.0.0.toml
│   └── S/
│       └── SA/
│           └── SARAHDEV/
│               ├── author.toml
│               └── async-utils/
│                   ├── meta.toml
│                   └── 0.5.0.toml
└── .github/
    └── workflows/
        └── validate.yml
```

### パス構造

```
packages/{1文字}/{2文字}/{AUTHOR}/{MODULE}/meta.json
```

例:
- `JOHNDOE/http` → `packages/J/JO/JOHNDOE/http/meta.json`
- `SARAHDEV/async-utils` → `packages/S/SA/SARAHDEV/async-utils/meta.json`

### 利点

- **スケーラビリティ**: 1ディレクトリに大量のファイルが集中しない
- **PR 競合回避**: 作者ごとにパスが分離
- **ブラウズ容易**: 作者別に整理
- **CPAN 方式**: 実績のある構造

---

## メタデータ形式

### packages/{A}/{AB}/{AUTHOR}/{MODULE}/meta.toml

```toml
name = "http"
author = "JOHNDOE"
description = "HTTP client library for Calcium"
repository = "https://github.com/johndoe/calcium-http"
license = "MIT"
keywords = ["http", "client", "network"]
latest = "1.1.0"
```

### packages/{A}/{AB}/{AUTHOR}/{MODULE}/{VERSION}.toml

バージョンごとのファイル一覧とハッシュ:

```toml
version = "1.1.0"
published = "2025-01-20"
entry = "mod.ca"
base_url = "https://raw.githubusercontent.com/johndoe/calcium-http/v1.1.0/"

[files]
"mod.ca" = "a1b2c3d4e5f6..."
"client.ca" = "b2c3d4e5f6a1..."
"server.ca" = "c3d4e5f6a1b2..."
"internal/utils.ca" = "d4e5f6a1b2c3..."
```

ファイルパス例:
- `packages/J/JO/JOHNDOE/http/meta.toml` (モジュール情報)
- `packages/J/JO/JOHNDOE/http/1.0.0.toml` (v1.0.0 のファイル一覧)
- `packages/J/JO/JOHNDOE/http/1.1.0.toml` (v1.1.0 のファイル一覧)

### ファイル解決の流れ

```
use "https://ca.land/JOHNDOE/http@1.1.0/client.ca"!;
```

1. **メタデータ取得**: `JOHNDOE/http/1.1.0.json`
2. **ファイル検索**: `files["client.ca"]` → ハッシュ値取得
3. **実体URL構築**: `base_url + "client.ca"`
   → `https://raw.githubusercontent.com/johndoe/calcium-http/v1.1.0/client.ca`
4. **ダウンロード & ハッシュ検証**
5. **キャッシュ保存 & ロード**

### インデックスファイル

#### index/all.toml（軽量インデックス）

モジュール検索用。名前と作者のみ。CI で自動生成。

```toml
version = "1"
updated = "2025-01-22T10:00:00Z"
count = 1523

[[modules]]
name = "http"
author = "JOHNDOE"
latest = "1.1.0"

[[modules]]
name = "json-parser"
author = "JOHNDOE"
latest = "2.0.0"

[[modules]]
name = "async-utils"
author = "SARAHDEV"
latest = "0.5.0"
```

#### index/{A-Z}.toml（作者別インデックス）

頭文字ごとの分割インデックス。CI で自動生成。

```toml
# index/J.toml

[authors.JOHNDOE]
modules = ["http", "json-parser"]
path = "J/JO/JOHNDOE"

[authors.JANEDOE]
modules = ["csv-reader"]
path = "J/JA/JANEDOE"
```

#### packages/.../author.toml（作者情報）

```toml
# packages/J/JO/JOHNDOE/author.toml
name = "JOHNDOE"
display_name = "John Doe"
github = "johndoe"
registered = "2025-01-15"
modules = ["http", "json-parser"]
```

### モジュール名の一意性

モジュール名は **作者スコープ** で一意:

```calcium
// 同じ名前でも作者が違えば別モジュール
use "https://ca.land/JOHNDOE/http@1.0.0/mod.ca"!;
use "https://ca.land/SARAHDEV/http@2.0.0/mod.ca"!;  // 別物
```

URL 構造:
```
https://{host}/{AUTHOR}/{MODULE}@{VERSION}/{PATH}
```

---

## モジュール登録フロー

### 1. 作者がモジュールを作成

```
my-calcium-lib/
├── mod.ca              # エントリーポイント
├── client.ca
├── server.ca
└── README.md
```

### 2. GitHub でタグを作成

```bash
git tag v1.0.0
git push origin v1.0.0
```

### 3. ファイルハッシュを計算

```bash
# 各ファイルの SHA256 を計算
sha256sum mod.ca client.ca server.ca
# a1b2c3d4...  mod.ca
# b2c3d4e5...  client.ca
# c3d4e5f6...  server.ca
```

### 4. レジストリに PR を送る

```bash
# calcium-lang/modules をフォーク
git clone https://github.com/yourname/modules
cd modules

# ディレクトリ作成（Y/YO/YOURNAME/my-lib/）
mkdir -p packages/Y/YO/YOURNAME/my-lib

# モジュール情報（初回のみ）
cat > packages/Y/YO/YOURNAME/my-lib/meta.toml << 'EOF'
name = "my-lib"
author = "YOURNAME"
description = "My awesome library"
repository = "https://github.com/yourname/my-calcium-lib"
license = "MIT"
keywords = ["utility"]
latest = "1.0.0"
EOF

# バージョン情報（ファイル一覧とハッシュ）
cat > packages/Y/YO/YOURNAME/my-lib/1.0.0.toml << 'EOF'
version = "1.0.0"
published = "2025-01-22"
entry = "mod.ca"
base_url = "https://raw.githubusercontent.com/yourname/my-calcium-lib/v1.0.0/"

[files]
"mod.ca" = "a1b2c3d4..."
"client.ca" = "b2c3d4e5..."
"server.ca" = "c3d4e5f6..."
EOF

git add .
git commit -m "Add YOURNAME/my-lib@1.0.0"
git push origin main
# → PR を作成
```

### 4. 自動検証（CI）

PR 時に GitHub Actions で検証:

- [ ] meta.json の形式チェック
- [ ] URL からファイルが取得できるか
- [ ] checksum が一致するか
- [ ] 名前の重複チェック
- [ ] ライセンス記載の確認

### 5. レビュー & マージ

メンテナがレビューしてマージ → レジストリに反映

---

## バージョン追加フロー

既存モジュールの新バージョン:

```bash
# 1. 自分のリポジトリで新タグ作成
git tag v1.1.0
git push origin v1.1.0

# 2. ファイルハッシュを計算
sha256sum mod.ca client.ca server.ca new-feature.ca

# 3. レジストリで新バージョンファイルを追加
cat > packages/Y/YO/YOURNAME/my-lib/1.1.0.toml << 'EOF'
version = "1.1.0"
published = "2025-01-25"
entry = "mod.ca"
base_url = "https://raw.githubusercontent.com/yourname/my-calcium-lib/v1.1.0/"

[files]
"mod.ca" = "..."
"client.ca" = "..."
"server.ca" = "..."
"new-feature.ca" = "..."
EOF

# 4. meta.toml の latest を更新
# latest = "1.1.0"

git commit -m "Add YOURNAME/my-lib@1.1.0"
# → PR を作成
```

---

## URL 解決

> **Note**: `ca.land` は仮のドメイン。実際は `calcium-modules.github.io` や取得可能なドメインを使用。

### URL 形式

```
https://{HOST}/{AUTHOR}/{MODULE}@{VERSION}/{PATH}
```

### ユーザーのコード

```calcium
use "https://ca.land/JOHNDOE/http@1.0.0/client.ca"!;  // 仮
// 実際は:
use "https://calcium-modules.github.io/JOHNDOE/http@1.0.0/client.ca"!;
```

### ホスト（仮）の役割

`ca.land` (仮) は GitHub Pages または Cloudflare Workers で:

1. `https://ca.land/JOHNDOE/http@1.0.0/client.ca` へのリクエストを受ける
2. `index/J.toml` から `JOHNDOE` のパスを取得
3. `packages/J/JO/JOHNDOE/http/1.0.0.toml` を参照
4. `base_url` + `files["client.ca"]` を取得
5. リダイレクト:
   ```
   → https://raw.githubusercontent.com/johndoe/calcium-http/v1.0.0/client.ca
   ```

### 静的ホスティング案

GitHub Pages で配信:

```
https://calcium-lang.github.io/modules/JOHNDOE/http@1.0.0/client.ca
                                    ↓ (redirect)
https://raw.githubusercontent.com/johndoe/calcium-http/v1.0.0/client.ca
```

---

## CLI サポート

### calcium publish（将来）

登録を簡略化するコマンド:

```bash
cd my-calcium-lib

# calcium.toml から meta.json を生成して PR 作成
calcium publish
```

```toml
# calcium.toml
[package]
name = "my-lib"
version = "1.0.0"
description = "My awesome library"
license = "MIT"
repository = "https://github.com/yourname/my-calcium-lib"
keywords = ["utility"]
```

---

## 検証ワークフロー

### .github/workflows/validate.yml

```yaml
name: Validate Package

on:
  pull_request:
    paths:
      - 'packages/**'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Validate meta.json
        run: |
          # JSON 形式チェック
          # 必須フィールドチェック
          # URL 到達確認
          # checksum 検証
          ./scripts/validate.sh

      - name: AI Review
        uses: calcium-lang/ai-review-action@v1
        with:
          package-path: ${{ github.event.pull_request.changed_files }}
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Auto Merge (if passed)
        if: steps.ai-review.outputs.status == 'pass'
        run: gh pr merge --auto --squash

      - name: Request Human Review (if flagged)
        if: steps.ai-review.outputs.status != 'pass'
        run: |
          gh pr comment --body "AI Review flagged this PR for human review."
          gh pr edit --add-label "needs-human-review"
```

---

## AI レビュー

PR は AI による自動レビューを基本とする。人間のレビューは例外的なケースのみ。

### レビュー基準

| 基準 | チェック内容 |
|------|-------------|
| **セキュリティ** | 危険なシステムコール、ファイル操作、ネットワーク通信の妥当性 |
| **悪意の有無** | 難読化コード、隠された機能、データ収集、バックドア |
| **目的の明示化** | README/説明文と実際のコードの一致、機能の明確さ |
| **ライセンス** | ライセンスの明記と妥当性 |

### ライセンスポリシー

**デフォルト MIT 強制ルール:**

- ライセンスが明記されていない場合 → **MIT ライセンスが自動適用**
- MIT 以外を希望する場合 → 明示的にライセンスを指定すること

```json
// meta.json
{
  "license": "MIT",           // 明記必須
  "license": "Apache-2.0",    // または他のライセンス
  "license": "UNLICENSED"     // プロプライエタリ（要説明）
}
```

**AI チェック項目:**

| 状態 | 判定 |
|------|------|
| `license` フィールドなし | → MIT を自動設定、警告付きで PASS |
| SPDX 識別子で明記 | → そのまま PASS |
| リポジトリに LICENSE ファイルなし | → 警告、meta.json の値を正とする |
| meta.json と LICENSE ファイルが矛盾 | → 人間レビューへ |
| 不明なライセンス文字列 | → 人間レビューへ |

**推奨ライセンス:**

- MIT - シンプル、広く使われている（デフォルト）
- Apache-2.0 - 特許条項あり
- BSD-3-Clause - MIT に近い
- GPL-3.0 - コピーレフト（依存関係に注意）

### AI レビューフロー

```
PR 作成
  ↓
CI: 形式チェック（JSON、URL 到達、checksum）
  ↓
AI レビュー:
  1. ソースコード取得（URL から）
  2. セキュリティスキャン
  3. 悪意パターン検出
  4. 目的と実装の整合性チェック
  ↓
┌─ PASS → 自動マージ
└─ FAIL/WARN → 人間レビューへエスカレーション
```

### AI レビュー結果

```json
{
  "status": "pass",
  "security": {
    "score": "safe",
    "notes": []
  },
  "malice": {
    "score": "none",
    "notes": []
  },
  "purpose": {
    "score": "clear",
    "declared": "HTTP client library",
    "actual": "HTTP client with GET/POST support",
    "match": true
  },
  "license": {
    "declared": "MIT",
    "file_found": true,
    "consistent": true,
    "auto_applied": false
  },
  "summary": "Safe HTTP client library with clear purpose and MIT license."
}
```

**ライセンス未指定時の結果例:**

```json
{
  "status": "pass",
  "license": {
    "declared": null,
    "file_found": false,
    "consistent": true,
    "auto_applied": true,
    "note": "No license specified. MIT license auto-applied."
  },
  "summary": "...(warning: MIT license was auto-applied)"
}
```

### エスカレーション条件

- セキュリティスコアが `warning` 以上
- 悪意スコアが `suspicious` 以上
- 目的の一致度が低い
- AI が判断を保留した場合

### 禁止パターン（自動拒否）

- `os.exec` 等の任意コマンド実行（正当な理由なし）
- 環境変数からの認証情報読み取り（非明示）
- 外部 URL へのデータ送信（非明示）
- Base64 エンコードされた実行コード
- 依存モジュールの動的ロード

---

## セキュリティ考慮

1. **checksum 必須** - 改ざん検知
2. **HTTPS のみ** - 中間者攻撃防止
3. **AI レビュー必須** - 悪意あるコードの自動検出
4. **作者認証** - GitHub アカウントで紐付け
5. **バージョン不変** - 公開後のバージョン内容変更禁止

---

## 段階的な実装

### Phase 1: 最小構成
- GitHub リポジトリ手動管理
- PR ベースの登録
- 静的 JSON ファイル

### Phase 2: 自動化
- CI による検証
- registry.json 自動生成
- GitHub Pages での配信

### Phase 3: CLI 統合
- `calcium search` コマンド
- `calcium publish` コマンド
- ローカルキャッシュとの連携

---

## 既存事例

| プロジェクト | 方式 |
|-------------|------|
| Homebrew | GitHub リポジトリ + PR |
| Deno (deno.land/x) | GitHub 連携 + webhook |
| Go (pkg.go.dev) | 自動クロール |
| Elm | GitHub リポジトリ + PR |

Calcium は **Homebrew / Elm 方式**（GitHub + PR）を採用。
