# 🚀 Iku

> Grammar-Aware Go Formatter: Structure through separation


Let your code breathe!

Iku is a grammar-based Go formatter that enforces consistent blank-line placement by AST node type.

## Philosophy

Code structure should be visually apparent from its formatting. Iku groups statements by grammatical type and separates them with blank lines, making the code flow easier to read at a glance.

## Rules

1. **Same AST type means no blank line**: Consecutive statements of the same type stay together
2. **Different AST type means blank line**: Transitions between statement types get visual separation
3. **Scoped statements get blank lines**: `if`, `for`, `switch`, `select` always have blank lines before them
4. **Top-level declarations are separated**: Functions, types, and variables at the package level get blank lines between them

## How It Works

Iku applies standard Go formatting (via [go/format](https://pkg.go.dev/go/format)) first ([formatter.go#L33](https://github.com/Fuwn/iku/blob/main/formatter.go#L33)), then adds its grammar-based blank-line rules on top. Your code gets `go fmt` output plus structural separation.

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
echo 'package main...' | iku

# Format and print to stdout
iku file.go

# Format in-place
iku -w file.go

# Format entire directory
iku -w ./...

# List files that need formatting
iku -l .

# Show diff
iku -d file.go
```

### Flags

| Flag | Description |
|------|-------------|
| `-w` | Write result to file instead of stdout |
| `-l` | List files whose formatting differs |
| `-d` | Display diffs instead of rewriting |
| `--comments` | Comment attachment mode: `follow`, `precede`, `standalone` |
| `--version` | Print version |

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
- `x := 1` and `y := 2` (both `AssignStmt`) stay together
- `var config` (`DeclStmt`) gets separated from assignments
- `defer` statements stay grouped together
- Each `if` statement gets a blank line before it (scoped statement)
- `go worker()` (`GoStmt`) is separated from the `if` above
- `return` (`ReturnStmt`) is separated from the `go` statement

### Top-Level Declarations

```go
// Before
package main
type Config struct {
    Name string
}
var defaultConfig = Config{}
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

var defaultConfig = Config{}

func main() {
    run()
}

func run() {
    process()
}
```

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

## AST Node Types

For reference, here are common Go statement types that Iku distinguishes:

| Type | Examples |
|------|----------|
| `*ast.AssignStmt` | `x := 1`, `x = 2` |
| `*ast.DeclStmt` | `var x = 1` |
| `*ast.ExprStmt` | `fmt.Println()`, `doSomething()` |
| `*ast.ReturnStmt` | `return x` |
| `*ast.IfStmt` | `if x > 0 { }` |
| `*ast.ForStmt` | `for i := 0; i < n; i++ { }` |
| `*ast.RangeStmt` | `for k, v := range m { }` |
| `*ast.SwitchStmt` | `switch x { }` |
| `*ast.SelectStmt` | `select { }` |
| `*ast.DeferStmt` | `defer f()` |
| `*ast.GoStmt` | `go f()` |
| `*ast.SendStmt` | `ch <- x` |

## License

This project is licensed under the [GNU General Public License v3.0](./LICENSE.txt).
