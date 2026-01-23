# バイトコードコンパイル仕様

## 概要

Calciumソースファイル(.ca)を事前にバイトコードファイル(.bone)にコンパイルし、実行時のパース・コンパイルを省略できる。

## 利点

1. **配布時のソース秘匿** - ソースコードを公開せずにプログラムを配布可能
2. **起動時間の短縮** - パース・コンパイル処理をスキップ
3. **事前最適化の保存** - 正規表現のコンパイル結果なども保存される

---

## コマンド

### コンパイル

```bash
# 基本形（出力は hello.bone）
calcium compile hello.ca

# 出力ファイル指定
calcium compile hello.ca -o output.bone
```

### 実行

```bash
# ソースを直接実行
calcium hello.ca

# コンパイル済みバイトコードを実行
calcium hello.bone

# run コマンドでも可
calcium run hello.ca
calcium run hello.bone
```

---

## ファイルフォーマット

### .bone (Calcium Bytecode) フォーマット

```
+-------------------+
| Magic: 4 bytes    |  "CALB"
+-------------------+
| Version: 2 bytes  |  major, minor
+-------------------+
| NumConstants: 4   |  定数の数（uint32, big-endian）
+-------------------+
| Constants...      |  各定数のシリアライズデータ
+-------------------+
| InsLen: 4 bytes   |  命令列の長さ（uint32）
+-------------------+
| Instructions...   |  バイトコード命令列
+-------------------+
```

### 定数のシリアライズ

各定数は型タグ(1バイト) + データで構成:

| 型タグ | 型 | データ形式 |
|--------|-----|------------|
| 0 | null | なし |
| 1 | bool | 1バイト (0 or 1) |
| 2 | int | 8バイト (int64, big-endian) |
| 3 | float | 8バイト (IEEE 754) |
| 4 | string | 4バイト長 + UTF-8データ |
| 5 | function | 後述 |
| 6 | regex | 後述 |

### Function のシリアライズ

```
[Name: string]
[NumParams: uint32]
[Params: string[]]
[NumLocals: uint32]
[IsEffect: byte]
[BodyLen: uint32]
[Body: bytes]
```

### Regex のシリアライズ

```
[Pattern: string]
[Flags: string]
```

※読み込み時に `regexp.Compile()` で再コンパイル

---

## バージョン互換性

- メジャーバージョンが異なる場合は読み込み不可
- マイナーバージョンの違いは許容（後方互換）

現在のバージョン: **1.0**

---

## 実装ファイル

| ファイル | 内容 |
|---------|------|
| `pkg/bytecode/serialize.go` | シリアライズ/デシリアライズ実装 |
| `pkg/bytecode/serialize_test.go` | テスト |
| `cmd/calcium/main.go` | CLI の compile/run コマンド |

---

## 制限事項

### シリアライズ不可の型

以下の型は定数プールに含まれないため問題なし:
- Closure（実行時に生成）
- Module（実行時にロード）
- Task/Handler/EventSource（非同期処理用、実行時生成）

### 動的機能

- `use` 文で読み込むモジュールは実行時に解決される
- モジュールの関数は実行時にバインドされる

---

## 使用例

### 基本的なワークフロー

```bash
# 開発時: ソースを直接実行
calcium app.ca

# リリース時: コンパイルして配布
calcium compile app.ca -o app.bone

# ユーザー: コンパイル済みを実行
calcium app.bone
```

### 正規表現の最適化

```calcium
// app.ca
use core.regex;
pattern = /^[a-z]+@[a-z]+\.[a-z]{2,}$/i;
email = "test@example.com";
regex.matches(email, pattern) |> io.println;
```

コンパイル時に正規表現がコンパイルされ、`.bone`ファイルに保存。
実行時は事前コンパイル済みの正規表現を即使用。

---

## 今後の拡張（検討中）

1. **デバッグ情報** - 行番号マッピングの保存
2. **圧縮** - gzip などによるファイルサイズ削減
3. **署名** - 改ざん検知用の署名
4. **最適化レベル** - `-O0`, `-O1`, `-O2` などのオプション
