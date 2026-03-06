# calcium-csv

RFC 4180 compliant CSV parser and builder for the [Calcium](https://github.com/ytnobody/calcium-lang) programming language.

## Installation

```
bone add ytnobody/csv
```

## Usage

```calcium
use csv;

// Parse CSV string into a 2D array
rows = csv.parse("name,age\nAlice,30\nBob,25");
// => [["name", "age"], ["Alice", "30"], ["Bob", "25"]]

// Parse CSV with header row into array of hashes
records = csv.parse_with_header("name,age\nAlice,30\nBob,25");
// => [{name: "Alice", age: "30"}, {name: "Bob", age: "25"}]

// Build CSV string from 2D array
data = [["name", "age"], ["Alice", "30"], ["Bob", "25"]];
csv_text = csv.stringify(data);
// => "name,age\r\nAlice,30\r\nBob,25"
```

## API

### `parse(text)`

Parses a CSV string and returns a 2D array of strings.

- Handles quoted fields (fields wrapped in double quotes)
- Handles commas and newlines within quoted fields
- Handles escaped double quotes (`""` within quoted fields)
- Supports both `\n` and `\r\n` line endings

```calcium
csv.parse("a,b,c\n1,2,3");
// => [["a", "b", "c"], ["1", "2", "3"]]
```

### `parse_with_header(text)`

Parses a CSV string using the first row as header keys, returning an array of hashes.

```calcium
csv.parse_with_header("name,age\nAlice,30");
// => [{name: "Alice", age: "30"}]
```

### `stringify(data)`

Converts a 2D array into a CSV string (RFC 4180 format, using `\r\n` line endings).

Fields containing commas, double quotes, or newlines are automatically quoted.
Double quotes within fields are escaped as `""`.

```calcium
csv.stringify([["hello, world"], ["say \"hi\""]]);
// => "\"hello, world\"\r\n\"say \"\"hi\"\"\""
```

## RFC 4180 Compliance

- Comma (`,`) field delimiter
- Double-quote (`"`) field enclosure
- Commas, newlines, and double-quotes within quoted fields
- Double-quote escaping with `""` inside quoted fields
- Both `\r\n` and `\n` line endings supported on input
- Output uses `\r\n` line endings

## License

MIT
