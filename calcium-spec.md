# Calcium 言語仕様書

Calcium は「算数をやってきた人が書ける」ことを目指した算数型プログラミング言語である。

## 設計思想

- 型ではなく「制約」で値を扱う
- 制御構文を最小限にする（if/else/while を持たない）
- 再代入を許さない
- 副作用を構文レベルで分離する
- 数学的な語彙で説明できる概念のみを使う

## コメント

単一行コメントと複数行コメントをサポートする。

```calcium
// 単一行コメント

/*
  複数行コメント
  複数行にわたって記述できる
*/
```

## 文の区切り

文の終わりにはセミコロン `;` を記述する。改行は空白として扱われるため、式は自由に複数行にまたがることができる。

```calcium
x = 5;
name = "calcium";

// 複数行にまたがる式
result = data
  |> transform
  |> validate;

items = [
  1,
  2,
  3
];

// 関数定義
func add(a, b) = a + b;
```

## 値

Calcium で扱える値は以下の6種類：

- 真偽値: `true`, `false`
- 数値: `42`, `3.14`, `-10`
  - 整数: `42`, `-10`
  - 浮動小数点: `3.14`, `-0.5`
  - 16進数: `0xFF`, `0x1A3F`
  - 2進数: `0b1010`, `0b11110000`
  - 科学的記数法: `1.5e10`, `2.5e-3`
  - 数値セパレータ: `1_000_000`, `0xFF_FF`
- 文字列: `"hello"`, `"世界"`
  - エスケープシーケンス: `\\`(バックスラッシュ), `\"`(ダブルクォート), `\n`(改行), `\t`(タブ), `\r`(キャリッジリターン)
- 配列: `[1, 2, 3]`, `["a", "b", "c"]`
- 関数: `(x) => x * 2`
- 結果型: `success(値)`, `failure(エラー)`

### 結果型（success/failure）

`success` と `failure` は組み込みの結果型である。

```calcium
// 結果を作成
success(42);         // 成功値
failure("error");    // 失敗値

// match でパターンマッチ
match result
  success(v) => v
  failure(e) => handle_error(e)
```

副作用関数 (`func!`) は暗黙的に結果型を返す：
- 成功時: `success(結果)`
- 失敗時: `failure(エラー)`

ユーザー定義のバリアント型（代数的データ型）は存在しない。

### null について

Calcium には `null` が存在しない。「値がないかもしれない」状況は `get` 関数と `success/failure` パターンで明示的に扱う。

```calcium
data = ["name", "田中"];

// 直接アクセス（キーが必ず存在する前提）
data.name;                   // "田中"
data.age;                    // エラー（キーが存在しない）

// 安全なアクセス
data |> get("name");         // success("田中")
data |> get("age");          // failure("key not found")
data |> get("age", 0);       // 0（デフォルト値を指定）

// JSON の null は failure として扱われる
json = parse_json('{"name": "田中", "age": null}');
json |> get("name");         // success("田中")
json |> get("age");          // failure("null")
json |> get("age", 0);       // 0
```

### 変数

変数は値に名前をつける。再代入はできない。

```calcium
x = 5
name = "calcium"
items = [1, 2, 3]
double = (x) => x * 2
```

再代入はエラー：

```calcium
x = 5
x = 10  // エラー
```

### 配列

配列は値のリストである。

```calcium
numbers = [1, 2, 3, 4, 5];
mixed = [1, "hello", [2, 3]];
```

配列リテラルでの連結（スペース区切り）：

配列リテラル内で要素をスペースで区切ると、配列が連結（フラット化）される。カンマで区切ると個別の要素として保持される。

```calcium
// スペース区切り = 連結
[1 2 3]              // [1, 2, 3]
[[1, 2] [3, 4]]      // [1, 2, 3, 4]
[[1] [2] [3]]        // [1, 2, 3]

// カンマ区切り = 個別要素
[1, 2, 3]            // [1, 2, 3]
[[1, 2], [3, 4]]     // [[1, 2], [3, 4]]（ネスト維持）

// 変数でも同様
a = [1, 2];
b = [3, 4];
[a b]                // [1, 2, 3, 4]（連結）
[a, b]               // [[1, 2], [3, 4]]（ネスト）

// 混在も可能
[a [5, 6] b]         // [1, 2, 5, 6, 3, 4]
```

インデックスアクセス（0始まり）：

```calcium
numbers = [10, 20, 30];

numbers[0];    // 10
numbers[2];    // 30
numbers[-1];   // 30（末尾から）
numbers[-2];   // 20（末尾から2番目）

// 範囲外アクセスはエラー
numbers[10];   // エラー

// 安全なアクセスは get を使う
numbers |> get(0);      // success(10)
numbers |> get(10);     // failure("index out of bounds")
numbers |> get(10, 0);  // 0（デフォルト値）
```

配列操作は `array` 名前空間で提供される：

```calcium
use array;

numbers = [10, 20, 30, 40, 50];

array.concat([1, 2], [3, 4]);    // [1, 2, 3, 4]
array.slice(numbers, 1, 3);      // [20, 30]（インデックス1から3未満）
array.slice(numbers, 2);         // [30, 40, 50]（インデックス2から末尾）
array.first(numbers);            // success(10)
array.last(numbers);             // success(50)
array.reverse(numbers);          // [50, 40, 30, 20, 10]
array.contains(numbers, 30);     // true
array.index_of(numbers, 30);     // success(2)
array.unique([1, 2, 2, 3]);      // [1, 2, 3]
```

`len` は標準で使用可能（配列と文字列の両方に対応）：

```calcium
len([1, 2, 3]);    // 3
len("hello");      // 5
```

配列の分解（変数束縛でスプレッド）：

```calcium
[head | tail] = [1, 2, 3];
// head = 1, tail = [2, 3]

[first, second | rest] = [1, 2, 3, 4, 5];
// first = 1, second = 2, rest = [3, 4, 5]

[a, b, c] = [1, 2, 3];
// a = 1, b = 2, c = 3

// 空配列や要素不足はエラー
[head | tail] = [];  // エラー
[a, b, c] = [1, 2];  // エラー
```

### ハッシュ（連想配列）

ハッシュは「キー, 値, キー, 値, ...」という形式の配列として表現する。

```calcium
person = ["name", "田中", "age", 25, "city", "東京"]
```

アクセス：

```calcium
person.name   // "田中"
person.age    // 25
```

キーと値の分離：

```calcium
keys(person)    // ["name", "age", "city"]
values(person)  // ["田中", 25, "東京"]
```

ハッシュの構築：

```calcium
k = ["a", "b", "c"]
v = [1, 2, 3]
hash(k, v)  // ["a", 1, "b", 2, "c", 3]
```

## 演算子

### 算術演算子

```calcium
+   // 加算（数値のみ）
-   // 減算
*   // 乗算
/   // 除算（常に浮動小数点）
%   // 剰余
**  // べき乗
```

`+` は数値専用。文字列連結には `concat` 関数を使用する：

```calcium
10 + 11                      // 21
"10" + "11"                  // エラー
concat("Hello", " ", "World") // "Hello World"
concat("value: ", 42)        // "value: 42"
```

### 比較演算子

```calcium
==  // 等価
!=  // 非等価
<   // 未満
>   // 超過
<=  // 以下
>=  // 以上
```

連鎖比較をサポートする：

```calcium
0 <= n <= 150   // 0 <= n && n <= 150 と等価
```

### 論理演算子

```calcium
&&  // 論理AND
||  // 論理OR
!   // 論理NOT
```

論理演算子は短絡評価を行う：
- `a && b`: `a` が偽なら `b` を評価せず偽を返す
- `a || b`: `a` が真なら `b` を評価せず真を返す

### スプレッド演算子

`...` は配列を引数列に展開する後置演算子。

```calcium
func add(x, y) = x + y;

// パイプラインでのスプレッド
[2, 3]... |> add           // add(2, 3) → 5

// 関数呼び出し内でのスプレッド
add([2, 3]...)             // add(2, 3) → 5

// 通常の引数と混在
func foo(a, b, c) = a + b + c;
foo(1, [2, 3]...)          // foo(1, 2, 3) → 6

// 変数にも適用可能
pair = [10, 20];
pair... |> add             // add(10, 20) → 30

// 配列連結と組み合わせ
[[1, 2] [3, 4]]... |> sum  // sum(1, 2, 3, 4)
```

### 演算子の優先順位

優先順位が高い順に以下の通り：

| 優先順位 | 演算子 | 説明 |
|---------|--------|------|
| 1 | `f(x)`, `obj.key`, `...` | 関数呼び出し、メンバアクセス、スプレッド |
| 2 | `-`, `!` | 単項演算子（負号、論理否定） |
| 3 | `**` | べき乗 |
| 4 | `*`, `/`, `%` | 乗算、除算、剰余 |
| 5 | `+`, `-` | 加算、減算 |
| 6 | `<`, `>`, `<=`, `>=` | 比較 |
| 7 | `==`, `!=` | 等価、非等価 |
| 8 | `&&` | 論理AND |
| 9 | `||` | 論理OR |
| 10 | `|>`, `!>` | パイプライン |
| 11 | `=` | 代入 |

## 制約

制約は値が満たすべき条件を定義する。型の代わりとして機能する。

### 制約の定義

```calcium
constraint Age(n) = 0 <= n <= 150;
constraint Email(s) = s |> matches(/^.+@.+\..+$/);
constraint Status(s) = s in ["active", "pending", "closed"];
constraint Positive(n) = n > 0;
constraint NonZero(n) = n != 0;
```

`:` は関数の引数に制約を付けるためにのみ使用する（後述）。

### 制約の評価タイミング

制約は可能な限りコンパイル時に検査される。静的に確定できない値は実行時に検査される。

```calcium
func divide(a, b: NonZero?) = a / b;

// リテラル値: コンパイル時に検査
divide(10, 0);       // コンパイルエラー

// 動的な値: 実行時に検査
x = read_input();
divide(10, x);       // x が 0 なら実行時エラー
```

実行時に制約違反が発生した場合の挙動：
- 純粋関数 (`func`) 内: プログラム停止（panic）
- 副作用関数 (`func!`) 内: `failure` を返す

### 制約の検査

制約名に `?` をつけて検査する。真偽値を返す。

```calcium
25 |> Age?        // true
200 |> Age?       // false
"test@example.com" |> Email?  // true
```

### 制約の組み合わせ

```calcium
constraint PositiveInt(n) = n > 0
constraint Under100(n) = n < 100
constraint Score(n) = n |> PositiveInt? && n |> Under100?
```

### 構造の制約

```calcium
constraint User(u) =
  u |> has("name") &&
  u |> has("age") &&
  u.age |> Age? &&
  u.name |> len |> (n => n > 0)
```

### 制約の内部実装

- 数値の制約: 比較演算で検証
- 文字列の制約: 正規表現で検証
- 列挙の制約: 含有チェック

## 関数

### 純粋関数

副作用を持たない関数は `func` で定義する。

```calcium
func add(a, b) = a + b

func double(x) = x * 2

func greet(name) = "Hello, " + name
```

### 副作用関数

副作用を持つ関数は `func!` で定義する。

```calcium
func! save(data) = ...
func! notify(user, message) = ...
func! fetch(url) = ...
```

### 引数の制約

```calcium
func divide(a, b: NonZero?) = a / b

func register(name: NonEmpty?, age: Age?) = ...
```

### 必須引数と残余引数

関数の引数は必須部分と残余部分に分けられる。

```calcium
func sum(| items) = items |> reduce(+)

func process(first, second | rest) = ...
```

呼び出し：

```calcium
sum(1, 2, 3, 4, 5)       // items = [1, 2, 3, 4, 5]
process(1, 2, 3, 4, 5)   // first = 1, second = 2, rest = [3, 4, 5]
process(1, 2)            // first = 1, second = 2, rest = []
process(1)               // エラー、必須引数が足りない
```

### 第一級関数

関数は値として扱える。

```calcium
double = (x) => x * 2;
triple = (x) => x * 3;

items |> map(double);
items |> map(triple);
```

### 関数のスコープ制約

関数は引数のみを参照できる。外部変数のキャプチャ（クロージャ）は禁止される。

```calcium
// OK: 引数のみを参照
func add(a, b) = a + b;
f = (x) => x * 2;

// NG: 外部変数を参照
n = 2;
f = (x) => x * n;   // エラー: n は引数ではない
```

すべての状態は引数として明示的に渡す：

```calcium
// NG: クロージャで n をキャプチャ
n = 2;
[1, 2, 3] |> map(x => x * n);

// OK: 関数に必要な値を引数で渡す
func multiply_by(n, x) = x * n;
[1, 2, 3] |> map(x => multiply_by(2, x));
```

### シャドーイング

内側のスコープで同名の変数を定義することができる（シャドーイング）。

```calcium
x = 10;
func foo(x) = x * 2;  // 引数 x は外側の x を隠す
foo(5);               // 10
```

関数は引数のみを参照できるため、シャドーイングによる混乱は起きない。

### 再帰

関数は自身を再帰的に呼び出すことができる。

```calcium
func factorial(n) =
  match n
    n <= 0 => 1
    _ => n * factorial(n - 1);

func sum_list(xs) =
  match len(xs)
    0 => 0
    _ => xs[0] + sum_list(array.slice(xs, 1));
```

末尾再帰最適化は実装の裁量とする（仕様では強制しない）。

## 分岐

### match

値によるパターン分岐を行う。

```calcium
func describe(x) =
  match x
    0 => "zero"
    n: Positive? => "positive"
    n: Negative? => "negative";
```

条件式を直接書くこともできる：

```calcium
func process(x) =
  match x
    0 < n < 100 => n * 2
    n > 0 => 100
    _ => 0;
```

ワイルドカード `_` はその他すべてにマッチする：

```calcium
func to_string(x) =
  match x
    0 => "zero"
    1 => "one"
    _ => "other";
```

```calcium
func handle(result) =
  match result
    success(v) => v |> process
    failure(e) => e |> log
```

### 網羅性チェック

match 式は網羅性がチェックされる：

- ワイルドカード `_` があれば網羅的
- `success`/`failure` を両方カバーしていれば網羅的
- それ以外はワイルドカードが必須（コンパイラが警告）

```calcium
// OK: ワイルドカードあり
match x
  0 => "zero"
  _ => "other"

// OK: success/failure を両方カバー
match result
  success(v) => v
  failure(e) => default

// 警告: 網羅的でない可能性
match x
  0 => "zero"
  1 => "one"
  // _ がないため警告
```

## 繰り返し

### map

配列の各要素に関数を適用する。

```calcium
numbers = [1, 2, 3, 4, 5]
doubled = numbers |> map(x => x * 2)  // [2, 4, 6, 8, 10]
```

### filter

条件を満たす要素だけを残す。

```calcium
numbers = [1, 2, 3, 4, 5]
evens = numbers |> filter(x => x % 2 == 0)  // [2, 4]
```

### reduce

配列を単一の値に集約する。

```calcium
numbers = [1, 2, 3, 4, 5];
total = numbers |> reduce((a, b) => a + b);  // 15
```

初期値を指定できる：

```calcium
[1, 2, 3] |> reduce((a, b) => a + b);        // 6（最初の要素が初期値）
[1, 2, 3] |> reduce((a, b) => a + b, 0);     // 6（初期値指定）

[] |> reduce((a, b) => a + b);               // エラー（空配列 + 初期値なし）
[] |> reduce((a, b) => a + b, 0);            // 0（初期値を返す）
```

### 非同期処理: core.async!

`core.async!` モジュールは非同期処理のためのプリミティブを提供する。
すべての非同期関連機能はこのモジュールの関数として提供される。

#### async.stay

`async.stay` は状態を持つイベント待機のための関数。`func!` の中でのみ使用できる。
Calcium で唯一、状態を扱える場所である。

```calcium
use core.async!
use core.io!
use core.schedule!

func! main() =
  task1 = async.spawn(() => fetch(url1))
  task2 = async.spawn(() => fetch(url2))

  async.stay(results: []) {
    task1.done
      |> async.expects((r) => {
        async.continue(results: [results [r]])
      })
      |> _.ready()

    task2.done
      |> async.expects((r) => {
        async.continue(results: [results [r]])
      })
      |> _.ready()

    schedule.timeout(5000)
      |> async.expects(() => {
        async.leave("timeout")
      })
      |> _.ready()

    async.all([task1, task2]).done
      |> async.expects(() => {
        async.leave(results)
      })
      |> _.ready()
  } !? {
    success(r) => log(r)
    failure(e) => log_error(e)
  };
```

#### async.spawn

`async.spawn` はバックグラウンドタスクを起動する副作用関数。`func!` 内でのみ使用可能。

```calcium
task = async.spawn(() => fetch(url))   // func! を spawn
task = async.spawn(() => compute(data)) // func も spawn 可
```

- `async.spawn` 自体は副作用（`func!` 内のみ）
- 引数の関数は pure (`func`) / impure (`func!`) どちらでも可
- 戻り値は `Task<a>` 型

#### async.expects と Handler<a>

`async.expects` はイベントハンドラを定義し、`Handler<a>` 型の値を返す。
パイプライン演算子でイベントソースと組み合わせて使用する。

```calcium
// イベントソース |> async.expects(ハンドラ関数) |> _.ready()
task.done
  |> async.expects((result) => {
    async.continue(results: [results [result]])
  })
  |> _.ready()

// 変数に束縛して後から有効化
handler = task.done
  |> async.expects((result) => {
    async.continue(results: [results [result]])
  })
handler.ready()
```

Handler は以下の状態を持つ：

```
dormant ──ready()──→ active ──async.cancel()──→ cancelled
                        ↑                           │
                        └─────────ready()───────────┘

active ──pause()──→ paused ──resume()──→ active
       ←─reset()─┘
```

Handler のメソッド：

| メソッド | 説明 |
|---------|------|
| `.ready()` | ハンドラを有効化。現在の `stay` に束縛される。戻り値は自身（チェーン可）。既に active なら no-op |
| `.reset()` | cancel + 再有効化。タイマーリセットなどに使用 |
| `.pause()` | 一時停止（イベントを無視） |
| `.resume()` | 一時停止から再開 |

Handler は `stay` の外で定義可能（dormant 状態）。`.ready()` を呼ぶと、その時点の `stay` に束縛され、`async.leave`/`async.continue` はその `stay` に対して作用する。

```calcium
// ファクトリ関数でハンドラを作成
make_timeout = (ms, msg) =>
  schedule.timeout(ms)
    |> async.expects(() => { async.leave(msg) })

func! main() =
  async.stay(count: 0) {
    timeout = make_timeout(5000, "timeout");
    timeout.ready();

    io.stdin
      |> async.expects((line) => {
        match line
          "cancel" => async.cancel(timeout); async.continue(count: count)
          _ => async.continue(count: count + 1)
      })
      |> _.ready()
  }
```

#### async.continue

状態を更新して待機に戻る。

```calcium
async.continue(count: count + 1)
async.continue(results: [results [r]], count: count + 1)
```

#### async.leave

`stay` ループを抜ける。デフォルトでは stay 内で spawn した全 Task が自動キャンセルされる。

```calcium
async.leave(value)                      // 全 Task を自動キャンセル
async.leave(value, keeping: [task1])    // 指定 Task のみ継続

// 全 Task を継続したい場合は自分で管理
tasks = [task1, task2, task3]
async.leave(value, keeping: tasks)
```

| 呼び出し | 動作 |
|---------|------|
| `async.leave(value)` | stay 内の全 Task を自動キャンセル |
| `async.leave(value, keeping: [tasks])` | 指定した Task のみ継続、残りはキャンセル |

#### async.all

複数のタスクの完了を待つ。

```calcium
async.all([task1, task2]).done
  |> async.expects(() => {
    async.leave(results)
  })
  |> _.ready()
```

#### async.cancel

Task または Handler をキャンセルする。

```calcium
async.cancel(handler)  // Handler のみ解除、紐づいた Task は継続
async.cancel(task)     // Task を中断 + 配下の全 Handler を解除
```

キャンセルの階層関係：

```
Task (async.spawn で生成)
  └── Handler (async.expects で生成)
        └── Handler
        └── Handler
```

- `async.cancel(handler)`: Handler のみ解除。Task は継続する
- `async.cancel(task)`: Task を中断し、その Task を待っている全 Handler も解除される

```calcium
task1 = async.spawn(() => fetch(url1))

async.stay(results: []) {
  h1 = task1.done
    |> async.expects((r) => { async.continue(results: [results [r]]) })
  h1.ready()

  h2 = task1.done
    |> async.expects((r) => { log(r); async.continue(results: results) })
  h2.ready()

  schedule.timeout(100)
    |> async.expects(() => {
      async.cancel(h1)  // h1 だけ解除、task1 と h2 は継続
      async.continue(results: results)
    })
    |> _.ready()

  schedule.timeout(500)
    |> async.expects(() => {
      async.cancel(task1)  // task1 中断 + h1, h2 両方解除
      async.leave("cancelled")
    })
    |> _.ready()
}
```

#### core.async! 関数一覧

| 関数 | 説明 |
|-----|------|
| `async.stay(state) { ... }` | イベントループ開始 |
| `async.spawn(fn)` | バックグラウンドタスク起動 → `Task<a>` |
| `async.expects(fn)` | イベントハンドラ生成 → `Handler<a>` |
| `async.continue(state)` | 状態更新して待機に戻る |
| `async.leave(value)` | ループ終了（全 Task キャンセル） |
| `async.leave(value, keeping: [...])` | 指定 Task のみ継続 |
| `async.cancel(target)` | Task/Handler をキャンセル |
| `async.all(tasks)` | 複数タスクをまとめる |

#### 並行性モデル

- シングルスレッド・イベントループ
- 複数の Handler は並行して待機するが、処理は逐次的
- 複数が同時に発火した場合、最初にマッチしたものを実行
- 1つのイベント処理が完了するまで次のイベントは処理しない

### I/O: core.io!

`core.io!` モジュールは入出力機能を提供する。

#### io.say / io.print

標準出力への出力。

```calcium
use core.io!

"Hello" |> io.say;    // Hello\n（改行あり）
"Hello" |> io.print;  // Hello（改行なし）

// 値は自動的に文字列に変換される
42 |> io.say;         // 42\n
[1, 2, 3] |> io.say;  // [1, 2, 3]\n
```

#### io.stdin / io.eof（イベントソース）

```calcium
use core.io!
use core.async!

async.stay(lines: []) {
  io.stdin
    |> async.expects((line) => {
      async.continue(lines: [lines [line]])
    })
    |> _.ready()

  io.eof
    |> async.expects(() => {
      async.leave(lines)
    })
    |> _.ready()
}
```

#### io.stdin

標準入力から1行読み込むイベントソース。

```calcium
io.stdin
  |> async.expects((line) => {
    // line に読み込んだ1行が束縛される
    process(line);
    async.continue(state: state)
  })
  |> _.ready()
```

#### io.eof

標準入力の終端（EOF）を検知するイベントソース。

```calcium
io.eof
  |> async.expects(() => {
    async.leave(result)
  })
  |> _.ready()
```

### スケジュール: core.schedule!

`core.schedule!` モジュールは時間ベースのイベントを提供する。

```calcium
use core.schedule!
use core.async!

async.stay(count: 0) {
  schedule.timeout(5000)
    |> async.expects(() => {
      async.leave("timeout")
    })
    |> _.ready()

  schedule.interval(1000)
    |> async.expects(() => {
      log(count);
      async.continue(count: count + 1)
    })
    |> _.ready()
}
```

#### schedule.timeout

指定ミリ秒後に1回だけ発火するイベントソース。

```calcium
schedule.timeout(5000)
  |> async.expects(() => {
    async.leave("timeout")
  })
  |> _.ready()
```

`handler.reset()` でタイマーをリセットできる：

```calcium
timeout = schedule.timeout(5000)
  |> async.expects(() => { async.leave("timeout") })
timeout.ready()

io.stdin
  |> async.expects((line) => {
    timeout.reset()  // タイマーをリセット（再度5秒後に発火）
    async.continue(state: state)
  })
  |> _.ready()
```

#### schedule.interval

指定ミリ秒ごとに繰り返し発火するイベントソース。

```calcium
schedule.interval(1000)
  |> async.expects(() => {
    log("tick");
    async.continue(state: state)
  })
  |> _.ready()
```

## パイプライン

### 純粋パイプライン `|>`

値を左から右へ流し、関数を適用していく。

```calcium
result = data
  |> transform
  |> validate
  |> format
```

### 副作用パイプライン `!>`

副作用を伴う処理を連鎖させる。`func!` の中でのみ使用できる。

```calcium
func! save_and_notify(data) =
  data
  !> save
  !> notify
  !? {
    success(_) => done()
    failure(e) => log_error(e)
  }
```

### エラー処理 `!?`

副作用パイプラインの終端で、必ずエラー処理を行う。

```calcium
data
!> save
!> notify
!? {
  success(result) => result |> log
  failure(err) => err |> log_error
}
```

`!>` を使ったら、必ず `!?` で終端しなければならない。

## 名前空間

### 定義

```calcium
// math.ca
namespace math;

func add(a, b) = a + b;
func multiply(a, b) = a * b;
constraint Positive(n) = n > 0;
```

### 使用

```calcium
// main.ca
use math;

result = 5 |> math.add(3) |> math.multiply(2);
```

特定の要素だけ取り込む：

```calcium
use math { add, Positive };

result = 5 |> add(3);
x: Positive? = 10;
```

ネストした名前空間：

```calcium
namespace util.string;

func trim(s) = ...;
func upper(s) = ...;
```

```calcium
use util.string { trim };

" hello " |> trim;
```

### モジュール解決

すべての `use` はエントリーポイントのディレクトリ（プロジェクトルート）からの相対パスで解決される。

```
project/
  common/
    utils.ca
  features/
    auth/
      login.ca    ← use common.utils; でOK
  main.ca         ← エントリーポイント
```

```calcium
// features/auth/login.ca
use common.utils;   // project/common/utils.ca を参照
```

解決順序：
1. プロジェクトルート（エントリーポイントのディレクトリ）からの相対パス
2. 標準ライブラリ

## エラーハンドリング

### コンパイル時エラー

制約違反はコンパイル時に検出される。

```calcium
func divide(a, b: NonZero?) = a / b

divide(10, 0)  // コンパイルエラー: 0 は NonZero? を満たさない
```

### 実行時エラー（副作用）

副作用に伴うエラーは `!?` で処理する。

```calcium
func! main() =
  data |> validate
  !> save_to_db
  !> notify_user
  !? {
    success(result) => result |> log
    failure(err) => err |> log_error
  }
```

## 完全な例

```calcium
// user.ca
namespace user

constraint Age(n) = 0 <= n <= 150
constraint Email(s) = s |> matches(/^.+@.+\..+$/)
constraint NonEmpty(s) = s |> len |> (n => n > 0)

constraint User(u) =
  u |> has("name") &&
  u |> has("age") &&
  u |> has("email") &&
  u.name |> NonEmpty? &&
  u.age |> Age? &&
  u.email |> Email?

func create(name, age, email) =
  ["name", name, "age", age, "email", email]

func validate(u) =
  match u |> User?
    false => failure("invalid user")
    true => success(u)
```

```calcium
// main.ca
use user
use core.io!

func! main() =
  input = ["name", "田中", "age", 25, "email", "tanaka@example.com"]

  result = input
    |> user.validate
    |> match
        success(u) => u
        failure(e) => leave e

  result
  !> save
  !> notify
  !? {
    success(_) => "Done!" |> io.say
    failure(e) => concat("Error: ", e) |> io.say
  }
```

## 標準関数

### グローバル関数（use 不要）

```calcium
len(xs)                  // 配列または文字列の長さ
get(collection, key)     // 安全なアクセス → success/failure
get(collection, key, default)  // デフォルト値付き
has(collection, key)     // キーまたはインデックスが存在するか → true/false
                         // ハッシュ: has(person, "name")
                         // 配列: has(numbers, 5)  // インデックス5が存在するか
concat(s1, s2, ...)      // 文字列連結
to_string(x)             // 任意の値を文字列に変換
to_num(s)                // 文字列を数値に変換 → success/failure
matches(s, regex)        // 正規表現マッチ → true/false
keys(hash)               // ハッシュのキー配列
values(hash)             // ハッシュの値配列
hash(keys, values)       // キー配列と値配列からハッシュを作成
```

### string 名前空間

```calcium
use string;

string.trim(s)           // 前後の空白を除去
string.upper(s)          // 大文字に変換
string.lower(s)          // 小文字に変換
string.split(s, sep)     // 分割 → 配列
string.join(xs, sep)     // 結合 → 文字列
```

### math 名前空間

```calcium
use math;

math.floor(n)            // 切り捨て
math.ceil(n)             // 切り上げ
math.round(n)            // 四捨五入
math.abs(n)              // 絶対値
math.max(a, b)           // 大きい方
math.min(a, b)           // 小さい方
```

### array 名前空間

配列操作の項を参照。

## 標準I/O

### 出力

出力は `core.io!` モジュールで提供される：

```calcium
use core.io!

"Hello" |> io.say;    // Hello\n（改行あり）
"Hello" |> io.print;  // Hello（改行なし）
```

### 入力

入力は `core.io!` モジュールの `io.stdin` イベントで取得する：

```calcium
use core.async!
use core.io!
use core.schedule!

async.stay(lines: []) {
  io.stdin
    |> async.expects((line) => {
      async.continue(lines: [lines [line]])
    })
    |> _.ready()

  schedule.timeout(1000)
    |> async.expects(() => {
      async.leave(lines)
    })
    |> _.ready()
} !? {
  success(result) => result |> map(line => line |> io.say)
  failure(e) => e |> io.say
};
```

詳細は「I/O: core.io!」セクションを参照。

## エントリーポイント

トップレベルのコードがそのまま実行される。`main` 関数は不要。

```calcium
use core.io!

// ファイルの内容がそのまま実行される
x = 10;
y = 20;
x + y |> io.say;
```

コマンドライン引数はグローバル変数 `args` で取得（プログラム名を含まない）：

```calcium
use core.io!

args[0];              // 最初の引数
len(args);            // 引数の数

// すべての引数を出力
args |> map(arg => arg |> io.say);
```

プログラム名は `program_name` で取得：

```calcium
use core.io!

program_name |> io.say;  // 実行中のプログラム名
```

## ファイル拡張子

`.ca` または `.calcium`

## 予約語

```
func func! constraint namespace use
match
map filter reduce
in has keys values hash len
success failure
return
```

注: 非同期処理関連の機能（`stay`, `spawn`, `expects`, `continue`, `leave`, `cancel` など）は言語キーワードではなく、`core.async!` モジュールの関数として提供される。同様に `timeout`, `interval` は `core.schedule!`、`stdin` は `core.io!` モジュールで提供される。
