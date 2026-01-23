# 正規表現コンパイル時最適化 仕様

## 概要

正規表現リテラルをバイトコードコンパイル時に事前コンパイルし、実行時のオーバーヘッドを排除する。

## 設計方針

1. **コンパイル時処理**: 正規表現は Calcium コンパイル時に Go の `regexp.Compile()` で事前コンパイル
2. **定数プール格納**: コンパイル済み `*regexp.Regexp` を定数として保存
3. **構文エラー早期検出**: 不正な正規表現はコンパイル時にエラー
4. **実行時ゼロコスト**: 実行時は事前コンパイル済みオブジェクトを即使用

---

## 構文

### 正規表現リテラル

```calcium
// 基本形
pattern = /^[a-z]+$/;

// フラグ付き
pattern = /hello/i;        // 大文字小文字無視
pattern = /^line$/m;       // マルチライン

// エスケープ
pattern = /https?:\/\//;   // スラッシュをエスケープ
pattern = /\d{3}-\d{4}/;   // 数字パターン
```

### フラグ

| フラグ | 意味 | Go での対応 |
|--------|------|------------|
| `i` | 大文字小文字無視 | `(?i)` プレフィックス |
| `m` | マルチライン | `(?m)` プレフィックス |
| `s` | `.` が改行にもマッチ | `(?s)` プレフィックス |

---

## レキサー変更

### 新トークン

```go
// token/token.go
const (
    REGEX  = "REGEX"   // /pattern/flags
)

type Token struct {
    Type    TokenType
    Literal string
    // 正規表現の場合、追加情報
    RegexPattern string  // パターン部分
    RegexFlags   string  // フラグ部分
    Line    int
    Column  int
}
```

### レキサーロジック

```go
// lexer/lexer.go
func (l *Lexer) NextToken() token.Token {
    // ...
    case '/':
        if l.isRegexContext() {
            return l.readRegex()
        }
        // 除算演算子
        return newToken(token.SLASH, l.ch, ...)
}

func (l *Lexer) isRegexContext() bool {
    // 直前のトークンを見て正規表現か除算かを判定
    // 正規表現になる文脈:
    //   - 文の先頭
    //   - = の後
    //   - ( の後
    //   - , の後
    //   - |> の後
    //   - 演算子の後
    // 除算になる文脈:
    //   - 識別子の後
    //   - 数値の後
    //   - ) の後
    //   - ] の後
}

func (l *Lexer) readRegex() token.Token {
    // /pattern/flags を読み取り
    // 1. 開始 / を消費
    // 2. 閉じ / まで読む（エスケープ \/ に注意）
    // 3. フラグ文字 [imsg]* を読む
    pattern := l.readUntilUnescaped('/')
    flags := l.readRegexFlags()
    return token.Token{
        Type:         token.REGEX,
        Literal:      "/" + pattern + "/" + flags,
        RegexPattern: pattern,
        RegexFlags:   flags,
    }
}
```

---

## パーサー変更

### AST ノード

```go
// ast/ast.go
type RegexLiteral struct {
    Token   token.Token
    Pattern string
    Flags   string
}

func (rl *RegexLiteral) expressionNode()      {}
func (rl *RegexLiteral) TokenLiteral() string { return rl.Token.Literal }
func (rl *RegexLiteral) String() string       { return "/" + rl.Pattern + "/" + rl.Flags }
```

### パース処理

```go
// parser/parser.go
func (p *Parser) parseRegexLiteral() ast.Expression {
    return &ast.RegexLiteral{
        Token:   p.curToken,
        Pattern: p.curToken.RegexPattern,
        Flags:   p.curToken.RegexFlags,
    }
}

func init() {
    // プレフィックス解析関数に登録
    p.registerPrefix(token.REGEX, p.parseRegexLiteral)
}
```

---

## Value 型

### 新しい型

```go
// value/value.go
const (
    TYPE_REGEX = "regex"
)

type Value struct {
    Type ValueType
    // ... 既存フィールド ...
    regex *regexp.Regexp  // コンパイル済み正規表現
}

func Regex(re *regexp.Regexp) Value {
    return Value{Type: TYPE_REGEX, regex: re}
}

func (v Value) AsRegex() *regexp.Regexp {
    return v.regex
}

func (v Value) String() string {
    if v.Type == TYPE_REGEX {
        return v.regex.String()
    }
    // ...
}
```

---

## コンパイラ変更

### コンパイル時処理

```go
// compiler/compiler.go
func (c *Compiler) Compile(node ast.Node) error {
    switch node := node.(type) {

    case *ast.RegexLiteral:
        // フラグをGo形式に変換
        goPattern := convertFlags(node.Pattern, node.Flags)

        // コンパイル時に正規表現をコンパイル
        re, err := regexp.Compile(goPattern)
        if err != nil {
            return fmt.Errorf("invalid regex /%s/: %s", node.Pattern, err)
        }

        // コンパイル済み正規表現を定数プールに追加
        idx := c.addConstant(value.Regex(re))
        c.emit(bytecode.OpConstant, idx)
    }
}

func convertFlags(pattern, flags string) string {
    // Calcium フラグを Go の (?flags) プレフィックスに変換
    prefix := ""
    for _, f := range flags {
        switch f {
        case 'i':
            prefix += "i"
        case 'm':
            prefix += "m"
        case 's':
            prefix += "s"
        }
    }
    if prefix != "" {
        return "(?" + prefix + ")" + pattern
    }
    return pattern
}
```

### エラー例

```
Error: invalid regex /[unclosed/: error parsing regexp: missing closing ]
  at line 5, column 10
```

---

## VM 変更

### 定数ロード

定数プールから `TYPE_REGEX` をロードするだけ（特別な処理不要）。

```go
case bytecode.OpConstant:
    constIndex := // ...
    c := vm.constants[constIndex]
    vm.push(c)  // regex もそのままプッシュ
```

---

## 組み込み関数

### matches(string, regex) → bool

```go
// vm/vm.go
func builtinMatches(args ...value.Value) value.Value {
    if len(args) != 2 {
        return value.Error("matches requires 2 arguments")
    }

    str := args[0]
    re := args[1]

    if str.Type != value.TYPE_STRING {
        return value.Error("first argument must be string")
    }
    if re.Type != value.TYPE_REGEX {
        return value.Error("second argument must be regex")
    }

    // 事前コンパイル済みなので即マッチング
    matched := re.AsRegex().MatchString(str.AsString())
    return value.Bool(matched)
}
```

### 使用例

```calcium
email = "test@example.com";
email |> matches(/^.+@.+\..+$/);  // true

// パイプラインで
input
    |> matches(/^\d{3}-\d{4}$/)
    |> validate;
```

---

## 追加の組み込み関数（将来）

| 関数 | 説明 |
|------|------|
| `matches(s, re)` | マッチするか → bool |
| `find(s, re)` | 最初のマッチを返す → success/failure |
| `find_all(s, re)` | 全マッチを配列で返す |
| `replace(s, re, replacement)` | 置換 |
| `split(s, re)` | 正規表現で分割 |
| `capture(s, re)` | キャプチャグループを配列で返す |

---

## 実装ステップ

### Phase 1: 基本実装

1. [x] `token.REGEX` トークン追加
2. [x] レキサーに正規表現リテラル読み取り実装
3. [x] `ast.RegexLiteral` ノード追加
4. [x] パーサーに正規表現パース実装
5. [x] `value.TYPE_REGEX` 追加
6. [x] コンパイラでコンパイル時コンパイル実装
7. [x] `matches` 組み込み関数実装
8. [x] テスト

### Phase 2: フラグ対応

1. [x] `i`, `m`, `s` フラグ実装
2. [x] フラグ → Go プレフィックス変換

### Phase 3: 追加関数

1. [x] `find`, `find_all`
2. [x] `replace`
3. [x] `split` (正規表現版)
4. [x] `capture`

---

## テストケース

```calcium
// 基本マッチ
"hello" |> matches(/hello/);           // true
"HELLO" |> matches(/hello/);           // false
"HELLO" |> matches(/hello/i);          // true

// パターン
"123-4567" |> matches(/^\d{3}-\d{4}$/); // true
"12-4567" |> matches(/^\d{3}-\d{4}$/);  // false

// エスケープ
"https://example.com" |> matches(/https?:\/\//);  // true

// コンパイルエラー
pattern = /[invalid/;  // コンパイル時エラー: missing closing ]
```

---

## パフォーマンス比較

```
実行時コンパイル方式:
  1000回ループ: ~50ms (毎回 regexp.Compile)

コンパイル時方式:
  1000回ループ: ~2ms (事前コンパイル済みを使用)

約25倍高速
```

---

## 注意点

### 除算との区別

```calcium
a = 10 / 2;        // 除算
b = /pattern/;     // 正規表現
c = x / y / z;     // 除算 / 除算
d = f(/pat/, /tern/);  // 関数引数に正規表現2つ
```

レキサーは直前のトークンを見て判定:
- 値（識別子、数値、`)`、`]`）の後 → 除算
- それ以外 → 正規表現

### 動的パターン

```calcium
// これはサポートしない（コンパイル時に確定しない）
pattern = "user-" + id;
matches(input, pattern);  // エラー: regex expected, got string

// 代わりに関数を使う
matches(input, /user-\d+/);  // OK
```

動的パターンが必要な場合は将来的に `regex(string)` 関数を追加検討。
