# 🚀 Iku

> Grammar-Aware Code Formatter: Structure through separation

Let your code breathe!

Iku is a grammar-based formatter that enforces consistent blank-line placement by statement and declaration type. It supports Go, JavaScript, TypeScript, JSX, and TSX.

## Philosophy

Code structure should be visually apparent from its formatting. Iku groups statements by grammatical type and separates them with blank lines, making the code flow easier to read at a glance.

## Rules

1. **Same kind means no blank line**: Consecutive bindings, calls, or assignments stay together.
2. **Different kinds get one blank line**: Bindings, assignments, calls, type declarations, imports, returns, and other grammatical roles are distinct.
3. **Bindings share a kind**: Go short declarations and explicit `var`/`const` declarations group together. JavaScript and TypeScript `const`, `let`, and `var` declarations also group together, including declaration modifiers.
4. **Multiline bodies get separation**: Control-flow bodies, functions, and type bodies get blank lines around them, without blank lines at the beginning or end of an enclosing block. Multiline initialisers alone do not count as bodies.
5. **Outer statements determine grouping**: Consecutive bindings inside a block stay together even when their values contain bodies. At top level, bindings containing multiline bodies get separation. Consecutive calls stay together even when they contain multiline callbacks.
6. **Comments attach to the following statement by default**: A comment group gets a blank line before it and none between it and the statement, except at the start of an enclosing block. Other attachment modes are configurable.
7. **Single-line definitions may cluster**: Consecutive single-line function and scoped type definitions group by default. This can be disabled in configuration.

## How It Works

For Go files, Iku applies standard Go formatting (via [go/format](https://pkg.go.dev/go/format)), classifies grammatical roles through the AST, and applies its blank-line rules. When its blank-line changes alter the layout, Iku formats the result again to keep alignment and required build-tag separators intact.

For JavaScript and TypeScript files (`.js`, `.ts`, `.jsx`, `.tsx`), Iku uses a heuristic line-based analyser that distinguishes declarations, assignments, and calls, and tracks delimiters and bodies. Strings, template literals, comments, and regular-expression literals are excluded from structural classification. This is not a complete JavaScript or TypeScript syntax parser.

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

# Format in-place
iku -w file.go
iku -w src/

# Format entire directory (Go, JS, TS, JSX, TSX)
iku -w .

# List files that need formatting
iku -l .

# Show diff
iku -d file.go
```

Directory walks skip hidden directories and `node_modules`. Explicit paths with other extensions are rejected, and standard input is formatted as Go.

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

All fields are optional. Omitted fields use their defaults; unknown fields and malformed JSON are reported as errors.

### `comment_mode`

Controls how comments interact with blank-line insertion. Default: `"follow"`.

| Mode | Behaviour |
|------|-----------|
| `follow` | Comments attach to the **next** statement. The blank line goes **before** the comment. |
| `precede` | Comments attach to the **previous** statement. The blank line goes **after** the comment. |
| `standalone` | Comments are independent. Blank lines are placed strictly by statement rules. |

### `group_single_line_functions`

When `true`, consecutive single-line function and scoped type definitions of the same kind are kept together without blank lines. Default: `true`. Set it to `false` to separate those definitions. Go output retains standard column alignment after regrouping.

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
- Reassignments such as `x = 2` have a different role from binding declarations
- `defer` statements stay grouped together
- Each `if` statement gets a blank line before it (scoped statement)
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
- `type ID int` and `type Name string` are unscoped type aliases, so they group together
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

In particular, `*ast.AssignStmt` represents both short declarations and
reassignments; Iku uses its token to distinguish those roles. Composite-literal
braces do not make a binding scoped, but a multiline function or type body does.

## License

Licensed under either of [Apache License, Version 2.0](LICENSE-APACHE) or
[MIT license](LICENSE-MIT) at your option.

Unless you explicitly state otherwise, any contribution intentionally submitted
for inclusion in this project by you, as defined in the Apache-2.0 license, shall
be dual licensed as above, without any additional terms or conditions.
