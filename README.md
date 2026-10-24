# ⛩️ Iku

> Grammar-Aware Blank-Line Formatter

Iku is a grammar-based formatter that enforces consistent blank-line placement by statement and declaration type. It supports Go, JavaScript, TypeScript, JSX, TSX, and Rust.

## Philosophy

Code structure should be visually apparent from its formatting. Iku groups statements by grammatical type and separates the groups with blank lines, so a reader can see where one step ends and the next begins.

## Rules

1. **Same kind means no blank line.** Consecutive bindings, calls, or assignments stay together.
2. **Different kinds get one blank line.** Bindings, assignments, calls, type declarations, imports, returns, and other grammatical roles are distinct.
3. **Bindings share a kind.** Go short declarations and explicit `var`/`const` declarations group together. JavaScript and TypeScript `const`, `let`, and `var` declarations also group together, including declaration modifiers, as do Rust `let`, `const`, and `static` bindings.
4. **Multiline bodies get blank lines around them.** This covers control-flow bodies, functions, and type bodies. Iku never adds a blank line at the start or end of an enclosing block. A multiline initialiser alone does not count as a body.
5. **Outer statements determine grouping.** Consecutive bindings inside a block stay together even when their values contain bodies. At top level, bindings containing multiline bodies get blank lines around them. Consecutive calls stay together even when they contain multiline callbacks.
6. **Comments attach to the following statement by default.** A comment group gets a blank line before it and none between it and the statement, except at the start of an enclosing block. [`comment_mode`](#comment_mode) selects other attachment modes.
7. **Single-line definitions may cluster.** Consecutive single-line function and scoped type definitions group by default. Set [`group_single_line_functions`](#group_single_line_functions) to `false` to separate them.

## How It Works

For Go files, Iku applies standard Go formatting (via [go/format](https://pkg.go.dev/go/format)), classifies grammatical roles through the AST, and applies its blank-line rules. When its blank-line changes alter the layout, Iku formats the result again to keep alignment and required build-tag separators intact.

For JavaScript and TypeScript files (`.js`, `.ts`, `.jsx`, `.tsx`), Iku uses a heuristic line-based analyser that distinguishes declarations, assignments, and calls, and tracks delimiters and bodies. The analyser ignores strings, template literals, comments, and regular-expression literals when classifying structure. It is not a complete JavaScript or TypeScript syntax parser.

For Rust files (`.rs`), Iku uses a similar heuristic analyser. It ignores strings, raw strings, character literals, and nested comments when classifying, but keeps lifetimes and labels. Attributes take the role of the item below them, and an attributed definition does not count as a single-line definition. Inner doc comments (`//!`) group with inner attributes rather than attaching to the next item. Iku classifies a block's final expression, including a closure's, as a return. Struct, enum, and union bodies are field lists, so their documentation stays packed and Iku removes blank lines between fields. Match arms are a kind of their own. Single-line arms group, and an arm with a multiline block body gets blank lines around it. Iku leaves brace-delimited macro bodies and `macro_rules!` definitions untouched. It is not a complete Rust syntax parser.

## Installation

```bash
go install github.com/Fuwn/iku@latest
```

Or run with Nix:

```bash
nix run github:Fuwn/iku
```

## Usage

```bash
# Format stdin
echo 'package main ...' | iku

# Format and print to stdout
iku file.go
iku component.tsx
iku main.rs

# Format in-place
iku -w file.go
iku -w src/

# Format entire directory (Go, JS, TS, JSX, TSX, Rust)
iku -w .

# List files that need formatting
iku -l .

# Show diff
iku -d file.go
```

Directory walks skip hidden directories, `node_modules`, and Cargo `target` directories marked with `CACHEDIR.TAG`. Iku rejects explicit paths with other extensions and formats standard input as Go.

### Flags

| Flag | Description |
|------|-------------|
| `-w` | Write result to file instead of stdout |
| `-l` | List files whose formatting differs |
| `-d` | Display unified diffs of formatting changes |
| `--version` | Print version |

## Configuration

Iku looks for `.iku.json` or `iku.json` in the current working directory.

```json
{
  "comment_mode": "follow",
  "group_single_line_functions": true
}
```

All fields are optional, and omitted fields use their defaults. Iku reports unknown fields and malformed JSON as errors.

### `comment_mode`

Controls how comments interact with blank-line insertion. The default is `"follow"`.

| Mode | Behaviour |
|------|-----------|
| `follow` | Comments attach to the next statement. The blank line goes before the comment. |
| `precede` | Comments attach to the previous statement. The blank line goes after the comment. |
| `standalone` | Comments attach to neither neighbour. Statement rules alone decide blank lines. |

### `group_single_line_functions`

When `true`, consecutive single-line function and scoped type definitions of the same kind stay together without blank lines. The default is `true`. Set it to `false` to separate those definitions. Go output keeps standard column alignment after regrouping.

```go
// group_single_line_functions = true (default)
func Base() string   { return baseDirectory }
func Config() string { return configFile }

// group_single_line_functions = false
func Base() string { return baseDirectory }

func Config() string { return configFile }
```

## Examples

### Before

```go
package main

func main() {
    x := 1
    y := 2
    var config = loadConfig()
    defer cleanup()
    defer closeDB()
    if err != nil {
        return err
    }
    if x > 0 {
        process(x)
    }
    go worker()
    return nil
}
```

### After

```go
package main

func main() {
    x := 1
    y := 2
    var config = loadConfig()

    defer cleanup()
    defer closeDB()

    if err != nil {
        return err
    }

    if x > 0 {
        process(x)
    }

    go worker()

    return nil
}
```

Notice how:
- `x := 1`, `y := 2`, and `var config` stay together because all three declare bindings
- A reassignment such as `x = 2` would not join them, because it is an assignment, not a binding
- `defer` statements stay grouped together
- Each `if` statement has a body, so it gets blank lines around it
- `go worker()` is separated from the `if` above
- `return` is separated from the `go` statement

### Top-Level Declarations

```go
// Before
package main
type Config struct {
    Name string
}
type ID int
type Name string
var defaultConfig = Config{}
var x = 1
func main() {
    run()
}
func run() {
    process()
}

// After
package main

type Config struct {
    Name string
}

type ID int
type Name string

var defaultConfig = Config{}
var x = 1

func main() {
    run()
}

func run() {
    process()
}
```

Notice how:
- `type Config struct` is scoped (has braces), so it gets a blank line
- `type ID int` and `type Name string` are unscoped type definitions, so they group together
- `var defaultConfig` and `var x` are unscoped, so they group together
- `func main()` and `func run()` are scoped, so each gets a blank line

### Switch Statements

```go
// Before
func process(x int) {
    result := compute(x)
    switch result {
    case 1:
        handleOne()
        if needsExtra {
            doExtra()
        }
    case 2:
        handleTwo()
    }
    cleanup()
}

// After
func process(x int) {
    result := compute(x)

    switch result {
    case 1:
        handleOne()

        if needsExtra {
            doExtra()
        }
    case 2:
        handleTwo()
    }

    cleanup()
}
```

## Grammatical Roles

Go AST node types inform classification, but do not determine grouping alone:

| Role | Examples |
|------|----------|
| Binding | `count := 1`, `var count = 1`, `const limit = 10` |
| Assignment | `count = 2`, `count += 1` |
| Call | `fmt.Println()`, `doSomething()` |
| Type Declaration | `type Count int`, `type Config struct { ... }` |
| Return | `return count` |
| Scoped Control Flow | `if`, `for`, `switch`, `select` |
| Deferred Call | `defer closeFile()` |
| Goroutine Launch | `go worker()` |
| Channel Send | `channel <- value` |

`*ast.AssignStmt` represents both short declarations and reassignments, and
Iku uses its token to tell those roles apart. Composite-literal
braces do not make a binding scoped, but a multiline function or type body does.

## License

Licensed under either of [Apache License, Version 2.0](LICENSE-APACHE) or
[MIT license](LICENSE-MIT) at your option.

Unless you explicitly state otherwise, any contribution intentionally submitted
for inclusion in this project by you, as defined in the Apache-2.0 license, shall
be dual licensed as above, without any additional terms or conditions.
