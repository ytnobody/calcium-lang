# Calcium リモートモジュール取り込み仕様 (Draft)

## 概要

Deno のモジュールシステムを参考に、URL ベースのモジュール取り込みを実現する。

## 設計方針

1. **URL ベース** - リモートモジュールは HTTPS URL で直接指定
2. **設定ファイル不要** - コード内で完結（必要なら import map で管理）
3. **グローバルキャッシュ** - 一度取得したモジュールは再利用
4. **バージョンは URL に含める** - `@1.0.0` のような形式

---

## 基本構文

### URL による直接インポート

```calcium
// HTTPS URL から直接インポート（作者スコープ）
use "https://ca.land/JOHNDOE/http@1.0.0/client.ca"!;
use "https://ca.land/SARAHDEV/json@2.0.0/mod.ca";

// GitHub raw URL（直接）
use "https://raw.githubusercontent.com/johndoe/calcium-http/v1.0.0/client.ca"!;
```

### URL 構造

```
https://ca.land/JOHNDOE/http@1.0.0/client.ca
         │       │       │   │       │
         │       │       │   │       └── ファイルパス
         │       │       │   └── バージョン
         │       │       └── モジュール名
         │       └── 作者名（大文字）
         └── ホスト（仮）
```

### 解決フロー

```
use "https://ca.land/JOHNDOE/http@1.0.0/client.ca"!;
```

```
1. メタデータ取得
   GET https://ca.land/JOHNDOE/http/1.0.0.toml

2. ファイル情報を検索
   1.0.0.toml の [files] から "client.ca" を探す
   → ハッシュ値 を取得
   → base_url + "client.ca" で実体URL を構築

3. 実体ファイル取得
   GET https://raw.githubusercontent.com/johndoe/calcium-http/v1.0.0/client.ca

4. ハッシュ検証
   SHA256(取得内容) == 1.0.0.toml のハッシュ値?
   → 一致: キャッシュに保存、ロード
   → 不一致: エラー（改ざんの可能性）
```

### ローカルモジュールとの使い分け

```calcium
// リモート（URL）- 作者/モジュール@バージョン
use "https://ca.land/JOHNDOE/http@1.0.0/client.ca"!;

// ローカル（従来通り）
use core.io!;             // 標準ライブラリ
use utils.helper;         // プロジェクト内 (src/utils/helper.ca)
```

**判定ルール**: `"` で始まる → URL、それ以外 → ローカル

---

## deps.ca パターン

依存を一箇所で管理する推奨パターン。

```calcium
// deps.ca - 依存の集約ファイル
pub use "https://ca.land/JOHNDOE/http@1.0.0/client.ca"! as http;
pub use "https://ca.land/JOHNDOE/json@2.0.0/mod.ca" as json;
pub use "https://ca.land/CALCIUM/std@1.0.0/async.ca"! as async;
```

```calcium
// main.ca - deps.ca 経由で使用
use deps { http, json };

result = http.get("https://api.example.com")
    |> json.parse;
```

**利点:**
- バージョン管理が一箇所で完結
- 更新時の変更箇所が最小限
- URL が散らばらない

---

## Import Map (calcium.imports.toml)

URL エイリアスを定義するオプション機能。

```toml
[imports]
"http/" = "https://ca.land/JOHNDOE/http@1.0.0/"
"json" = "https://ca.land/JOHNDOE/json@2.0.0/mod.ca"
"std/" = "https://ca.land/CALCIUM/std@1.0.0/"
```

```calcium
// import map 使用時
use "http/client.ca"!;   // → https://ca.land/JOHNDOE/http@1.0.0/client.ca
use "json";              // → https://ca.land/JOHNDOE/json@2.0.0/mod.ca
use "std/async.ca"!;     // → https://ca.land/CALCIUM/std@1.0.0/async.ca
```

### 実行時の指定

```bash
calcium run --import-map=calcium.imports.toml src/main.ca
```

---

## キャッシュ

### グローバルキャッシュ

```
~/.calcium/
└── cache/
    └── https/
        └── ca.land/
            ├── JOHNDOE/
            │   ├── http@1.0.0/
            │   │   └── client.ca
            │   └── json@2.0.0/
            │       └── mod.ca
            └── CALCIUM/
                └── std@1.0.0/
                    └── async.ca
```

### キャッシュ操作

```bash
# 依存を事前キャッシュ
calcium cache src/main.ca
calcium cache deps.ca

# キャッシュ情報表示
calcium cache --info

# キャッシュクリア
calcium cache --clear
```

---

## Lock ファイル (calcium.lock)

整合性検証用。`calcium cache` または初回実行時に自動生成。TOML 形式。

```toml
version = "1"
generated = "2025-01-22T10:00:00Z"

[modules."JOHNDOE/http@1.0.0"]
meta_url = "https://ca.land/JOHNDOE/http/1.0.0.toml"
meta_sha256 = "fff000..."

[modules."JOHNDOE/http@1.0.0".files]
"client.ca" = { url = "https://raw.githubusercontent.com/johndoe/calcium-http/v1.0.0/client.ca", sha256 = "a1b2c3..." }

[modules."JOHNDOE/json@2.0.0"]
meta_url = "https://ca.land/JOHNDOE/json/2.0.0.toml"
meta_sha256 = "eee111..."

[modules."JOHNDOE/json@2.0.0".files]
"mod.ca" = { url = "https://raw.githubusercontent.com/johndoe/calcium-json/v2.0.0/mod.ca", sha256 = "b2c3d4..." }
```

**Lock ファイルの役割:**
- メタデータ自体のハッシュも記録（改ざん防止）
- 実際に使用したファイルのみ記録
- CI 環境で完全な再現性を保証

```bash
# lock ファイルで整合性チェック
calcium run --lock=calcium.lock src/main.ca

# lock ファイルを更新
calcium cache --lock=calcium.lock --lock-write src/main.ca
```

---

## モジュール解決順序

```calcium
use foo.bar;              // ローカル
use "https://.../mod.ca"; // リモート
```

**ローカル (`use foo.bar;`)**
1. プロジェクト内: `src/foo/bar.ca`
2. 標準ライブラリ: `core.*`

**リモート (`use "https://...";`)**
1. Import Map でマッピング（あれば）
2. キャッシュを確認
3. なければネットワークから取得

---

## CLI コマンド

### calcium run

```bash
# 基本実行（必要なモジュールを自動取得）
calcium run src/main.ca

# import map 指定
calcium run --import-map=calcium.imports.json src/main.ca

# lock ファイルで整合性チェック
calcium run --lock=calcium.lock src/main.ca

# ネットワークアクセス禁止（キャッシュのみ使用）
calcium run --cached-only src/main.ca
```

### calcium cache

```bash
# 依存を事前キャッシュ
calcium cache src/main.ca

# キャッシュ情報
calcium cache --info

# キャッシュクリア
calcium cache --clear

# lock ファイル生成
calcium cache --lock=calcium.lock --lock-write src/main.ca
```

---

## 完全な例

### プロジェクト構成

```
my-project/
├── calcium.imports.toml   # (オプション) import map
├── calcium.lock           # (自動生成) TOML形式
├── deps.ca                # 依存集約
└── src/
    └── main.ca
```

### deps.ca

```calcium
// deps.ca
pub use "https://ca.land/JOHNDOE/http@1.0.0/client.ca"! as http;
pub use "https://ca.land/JOHNDOE/json@2.0.0/mod.ca" as json;
```

### src/main.ca

```calcium
// src/main.ca
use core.io!;
use deps { http, json };

data = http.get("https://api.example.com/users")
    |> json.parse;

io.println(data);
```

### 実行

```bash
# 初回実行（依存を自動取得してキャッシュ）
calcium run src/main.ca

# lock ファイル生成（CI 用）
calcium cache --lock=calcium.lock --lock-write src/main.ca

# CI 環境
calcium run --lock=calcium.lock --cached-only src/main.ca
```

---

## 実装ステップ

1. **パーサー拡張**: `use "https://..."` 構文の対応
2. **TOML パーサー**: メタデータ・Lock・Import Map の読み書き
3. **HTTP クライアント**: URL からファイル取得
4. **SHA256**: ハッシュ計算・検証
5. **キャッシュ管理**: `~/.calcium/cache/` への保存・読み込み
6. **モジュール解決**: URL → メタデータ → 実体URL → キャッシュ

---

## 未決定事項

- [ ] `pub use ... as` 構文のパーサー対応
- [ ] キャッシュディレクトリの環境変数 (`CALCIUM_DIR`)
- [ ] リダイレクト対応（何回まで追跡するか）
- [ ] タイムアウト設定
- [ ] プライベート URL の認証（Authorization ヘッダ）
