# Project Guidelines

- This is a cellular automata simulation desktop app written in Go.
- Graphics rendering will use Ebitengine.
- When writing code, always optimize for robustness and keep the Go code idiomatic, leveraging the language's strengths and principles.
- Keep technical documentation up to date whenever adding or changing code.

---

Here’s the quick mental model: **idiomatic Go** favors *clarity over cleverness*, *composition over inheritance*, *small pieces wired simply*, and *straight-line error handling*. It’s practical, explicit, and predictable.

## Core principles (fast scan)

* **Simple > clever.** Prefer straightforward control flow and types.
* **Small interfaces.** Define behavior you actually need (“interface as seam,” not as default).
* **Explicit errors.** Return `error`, wrap with `%w`, avoid panics except for programmer bugs.
* **Zero values work.** Design types that are usable without mandatory init.
* **Avoid unnecessary abstraction.** Don’t add layers until you have two+ concrete callers.
* **Concurrency is explicit.** Use `context.Context`, channels, and goroutines with clear ownership; stop them cleanly.
* **Package over classes.** Group by domain; keep packages small and focused.
* **Value semantics by default.** Use pointers when you need mutation/sharing; otherwise pass values.
* **Naming is plain.** Short, concrete names; exported only when needed.

---

## Tiny good vs bad snippets

### 1) Errors: straight-line & wrapped

**Bad**

```go
res, err := doThing()
if err != nil { return err } // loses cause
```

**Good**

```go
res, err := doThing()
if err != nil {
    return fmt.Errorf("doThing: %w", err)
}
```

### 2) Small, focused interfaces

**Bad**

```go
type Storage interface {
    Save([]byte) error
    Load(string) ([]byte, error)
    Delete(string) error
    List() ([]string, error)
}
```

**Good**

```go
type Reader interface { Load(string) ([]byte, error) }
type Writer interface { Save([]byte) error }
```

*(Define only what the caller needs; compose later.)*

### 3) Don’t return nil slices/maps when empty handling is expected

**Bad**

```go
func Keys(m map[string]int) []string { return nil }
```

**Good**

```go
func Keys(m map[string]int) []string { return []string{} }
```

*(Callers can range over empty slice without nil checks.)*

### 4) Zero-values should be useful

**Bad**

```go
type Counter struct { n int; mu *sync.Mutex }
func NewCounter() *Counter { return &Counter{mu: &sync.Mutex{}} }
```

**Good**

```go
type Counter struct {
    n  int
    mu sync.Mutex // zero value is ready
}
```

### 5) Context for cancellation, not data

**Bad**

```go
func Fetch(ctx context.Context, url, user string) {} // smuggling params
```

**Good**

```go
func Fetch(ctx context.Context, url string) {} // use ctx for deadlines/cancel only
```

### 6) Avoid getters/setters when fields can be plain

**Bad**

```go
func (c *Config) GetPort() int { return c.port }
```

**Good**

```go
type Config struct { Port int } // exported if external packages need it
```

### 7) Prefer composition to inheritance-like hierarchies

**Bad**

```go
type Animal struct { Name string }
type Dog struct { Animal }
```

**Good**

```go
type Dog struct {
    Name string
}
```

*(Use embedding only when it genuinely simplifies method sets.)*

### 8) Goroutine lifecycle: own your goroutines

**Bad**

```go
func start() {
    go worker() // never stopped
}
```

**Good**

```go
func start(ctx context.Context) {
    go func() {
        defer close(done)
        for {
            select {
            case <-ctx.Done():
                return
            case job := <-jobs:
                _ = handle(job)
            }
        }
    }()
}
```

### 9) Channels: range and close semantics

**Bad**

```go
for {
    v, ok := <-ch
    if !ok { break }
    use(v)
}
```

**Good**

```go
for v := range ch {
    use(v)
}
```

*(Producer closes, consumers range.)*

### 10) Receiver choices: pointer vs value

**Bad**

```go
func (b Box) Put(x int) { b.v = x } // no effect
```

**Good**

```go
func (b *Box) Put(x int) { b.v = x }        // mutates
func (b Box) Value() int { return b.v }     // read-only by value is fine
```

### 11) Errors over booleans/sentinels

**Bad**

```go
if !ok { return false }
```

**Good**

```go
if err != nil { return fmt.Errorf("parse config: %w", err) }
```

### 12) Avoid over-abstracting data structures

**Bad**

```go
type Node[T any] struct{ /* generic tree for one use */ }
```

**Good**

```go
type Grid struct { W, H int; Cells []byte } // fits the domain plainly
```

---

## Smell checks (ask yourself)

* *Can a newcomer read this in one pass?*
* *Does zero value work?*
* *Is the interface the smallest surface that unblocks the caller?*
* *Do callers need to know about goroutines/mutexes here? (Hide complexity.)*
* *Are errors actionable and wrapped?*
* *Could this package be split to reduce exported names?*
