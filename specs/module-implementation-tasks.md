# モジュールシステム実装タスク

## 概要

仕様に基づき、以下の実装が必要。

---

## 1. レジストリ運用ツール（GitHub Actions）

### validate.yml - PR 検証

```yaml
# .github/workflows/validate.yml
on:
  pull_request:
    paths: ['packages/**']
```

**実装内容:**
- [ ] TOML 形式チェック（meta.toml, {version}.toml）
- [ ] 必須フィールド検証
- [ ] URL 到達確認（base_url + 各ファイル）
- [ ] SHA256 検証（実体ファイル取得→ハッシュ突合）
- [ ] ライセンス検証（未指定→MIT 自動適用）
- [ ] AI レビュー呼び出し

### build-index.yml - インデックス再構築

```yaml
# .github/workflows/build-index.yml
on:
  push:
    branches: [main]
    paths: ['packages/**']
```

**実装内容:**
- [ ] packages/ を走査
- [ ] index/all.toml 生成
- [ ] index/{A-Z}.toml 生成
- [ ] author.toml の modules リスト更新

---

## 2. レジストリ用スクリプト

### scripts/validate.sh

```bash
#!/bin/bash
# PR で追加/変更された TOML を検証
```

**機能:**
- TOML パース（tomlq や Go ツール）
- HTTP リクエスト（curl）
- SHA256 計算（sha256sum）

### scripts/build-index.sh

```bash
#!/bin/bash
# packages/ からインデックスを再構築
```

### scripts/ai-review.sh

```bash
#!/bin/bash
# AI API を呼び出してレビュー実行
```

---

## 3. Calcium CLI 拡張

### calcium cache

```
calcium cache <file.ca>           # 依存を事前取得
calcium cache --info              # キャッシュ情報
calcium cache --clear             # キャッシュクリア
```

**実装（Go）:**
- [ ] `use "https://..."` 文をパース
- [ ] メタデータ TOML 取得
- [ ] ファイル一覧からハッシュ取得
- [ ] 実体ダウンロード & ハッシュ検証
- [ ] `~/.calcium/cache/` に保存

### calcium run 拡張

```
calcium run --import-map=calcium.imports.toml src/main.ca
calcium run --lock=calcium.lock src/main.ca
calcium run --cached-only src/main.ca
```

**実装（Go）:**
- [ ] import map 読み込み・URL 変換
- [ ] lock ファイル読み込み・ハッシュ検証
- [ ] --cached-only でネットワーク無効化

---

## 4. 必要なライブラリ/モジュール

### Go 側（CLI）

| パッケージ | 用途 |
|-----------|------|
| `github.com/BurntSushi/toml` | TOML パース |
| `crypto/sha256` | ハッシュ計算 |
| `net/http` | HTTP クライアント |
| `os` | キャッシュディレクトリ管理 |

### シェルスクリプト側（CI）

| ツール | 用途 |
|--------|------|
| `tomlq` or `yj` | TOML パース（jq 的に） |
| `curl` | HTTP リクエスト |
| `sha256sum` | ハッシュ計算 |
| `gh` | GitHub CLI（PR 操作） |

---

## 5. 実装順序

### Phase 1: 最小限のレジストリ

1. GitHub リポジトリ `calcium-lang/modules` 作成
2. ディレクトリ構造セットアップ
3. `scripts/validate.sh` 実装
4. `validate.yml` 設定
5. サンプルモジュール登録（手動）

### Phase 2: Calcium CLI 対応

1. TOML パーサー追加（Go）
2. `use "https://..."` パーサー対応
3. HTTP クライアント実装
4. キャッシュ管理実装
5. `calcium cache` コマンド実装

### Phase 3: 自動化

1. `build-index.yml` 実装
2. AI レビュー統合
3. `calcium run` のオプション拡張
4. Lock ファイル生成

### Phase 4: 利便性向上

1. `calcium search` コマンド
2. `calcium publish` コマンド（メタデータ生成補助）
3. エラーメッセージ改善
4. ドキュメント整備

---

## 6. ファイル配置（予定）

```
calcium/
├── cmd/calcium/
│   └── main.go              # CLI エントリ（cache, run 拡張）
├── pkg/
│   ├── module/
│   │   ├── resolver.go      # モジュール解決
│   │   ├── cache.go         # キャッシュ管理
│   │   ├── fetch.go         # HTTP 取得 + ハッシュ検証
│   │   └── toml.go          # TOML パース
│   └── ...
└── specs/
    ├── module-import-spec.md
    └── module-registry-spec.md
```

---

## 7. テスト計画

### ユニットテスト

- TOML パース
- SHA256 計算
- URL → キャッシュパス変換
- Import Map 適用

### 統合テスト

- メタデータ取得 → ファイル取得 → ハッシュ検証 → キャッシュ保存
- Lock ファイル生成・読み込み
- --cached-only での実行

### E2E テスト

- 実際のレジストリからモジュール取得
- PR 検証ワークフロー
