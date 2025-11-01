**QUESTION**:
Regarding the test called `TestDefaultSystem_RequireDir_NotADir`: isn't this a great example of how returning interfaces from methods in Go is beneficial? The general mantra in Go is "accept interfaces, return structs". But from this example, it seems that returning structs make the code less testable. If `os.FileInfo` was a struct, I would not have been able to write this test, and I would have to have used a real `os.FileInfo` (which I guess would require me to create a real file on my system).

What are your thoughts on this mantra? Why could it be beneficial to return structs instead of interfaces? Right now I don't see any positive points

TODO/NOTE: I need to read the following and think about it along with this response: https://tutorialedge.net/golang/accept-interfaces-return-structs/

---

Excellent observation! You've identified a real tension in Go design philosophy. Let me break down the nuanced reality:

## The Mantra Is Contextual, Not Absolute

**"Accept interfaces, return structs"** is a guideline, not a law. The full picture is more subtle:

### When to Return Interfaces (Like `os.FileInfo`)

**1. Abstraction Over Implementation Details**
```go
// Good: Caller doesn't need to know the concrete type
func Stat(name string) (FileInfo, error)
```


The caller only cares about `IsDir()`, `Size()`, etc. - not whether it's a file, symlink, or pipe. The concrete type would leak implementation details.

**2. Multiple Possible Implementations**
```go
// os.FileInfo can represent files, directories, symlinks, devices, etc.
// Returning a concrete struct would limit this flexibility
```


**3. Testing (Your Point!)**
Yes, returning interfaces makes code testable without filesystem I/O, network calls, etc.

### When to Return Structs

**1. Known, Stable Types**
```go
// Good: Time is a concrete value type
func Now() time.Time

// Bad: Would be over-abstraction
func Now() TimeProvider
```


**2. Performance**
Interfaces have runtime overhead (dynamic dispatch, heap allocation for small values). For hot paths:
```go
// Returning struct avoids interface boxing
func ParseInt(s string) (int, error)  // not (Integer, error)
```


**3. Prevent Unwanted Polymorphism**
```go
// You want callers to use THIS specific type, not substitute their own
type Config struct { /* ... */ }
func LoadConfig() (*Config, error)  // not (ConfigProvider, error)
```


**4. Value Semantics**
```go
// You want copy-by-value behavior, clear equality semantics
type Point struct { X, Y int }
func NewPoint(x, y int) Point  // not PointInterface
```


## Your Specific Case: `os.FileInfo`

The Go stdlib **intentionally** returns `os.FileInfo` interface because:

1. **Cross-platform**: Linux, Windows, macOS have different filesystem metadata. The interface hides these differences.

2. **Multiple sources**: Can represent local files, network files, virtual filesystems, in-memory files, etc.

3. **Testability**: Exactly your point - you can mock it without touching the filesystem.

## The Real Guideline Should Be:

> **"Accept interfaces, return concrete types UNLESS you need abstraction, multiple implementations, or testability at the boundary"**

### Decision Tree:

```
Should I return an interface?
├─ Do I have multiple implementations? → YES: Interface
├─ Is this a boundary/seam (I/O, external systems)? → YES: Interface
├─ Do consumers need to mock this? → YES: Interface
├─ Is this a simple value type? → NO: Struct
├─ Do I want strong type safety? → NO: Struct
└─ Default → Struct (can always wrap later)
```


## Why the Mantra Exists (Despite Exceptions)

**1. Premature Abstraction**
```go
// Bad: Over-engineering
type UserGetter interface {
    GetUser() User
}

func NewUserService() UserGetter { /* ... */ }
```


You've created an interface with ONE implementation, adding complexity for no benefit.

**2. Interface Pollution**
```go
// Bad: Every struct gets a useless interface
type IUser interface { GetName() string; GetEmail() string }
type User struct { Name, Email string }
```


This is "Java-style" Go - creates maintenance burden.

**3. Compile-Time Safety**
Returning structs gives you:
- Clear API surface
- Better IDE autocomplete
- Refactoring safety
- No surprise behavior from alternative implementations

## In Your Code

Your `stdlib` interface is a **perfect** use of returning interfaces:

```go
type stdlib interface {
    Stat(name string) (os.FileInfo, error)  // Returns interface ✓
}
```


This is a **seam** for testing - exactly where interfaces shine. You're wrapping the standard library's I/O boundary.

## Bottom Line

You're right that `os.FileInfo` being an interface is crucial for testability. But this doesn't invalidate the general guideline - it shows that **I/O boundaries, external dependencies, and abstraction points** are exactly where you *should* return interfaces.

The mantra really means: "Don't create interfaces just because you can - create them when you need them." And for stdlib I/O operations, you definitely need them.

Your TODO comment should probably say:
```go
// NOTE: os.FileInfo is correctly an interface because:
// 1. It represents multiple filesystem implementations
// 2. It's an I/O boundary that needs mocking for tests
// 3. Callers don't need to know the concrete type
// This is an exception to "return structs", not a violation of it.
```
