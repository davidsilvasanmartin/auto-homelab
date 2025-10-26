# Using `slog` instead of `fmt.Print...` functions

The `slog` package (available since Go 1.21) provides structured logging with different log levels. Here's how to migrate from `fmt.Print...` functions:

## Key Concepts

1. **Import the package**: `import "log/slog"`
2. **Choose a logger**: Use the default logger or create a custom one
3. **Use appropriate log levels**:
    - `slog.Info()` - for informational messages (replaces most `fmt.Println`)
    - `slog.Error()` - for errors (replaces `fmt.Fprintln(os.Stderr, ...)`)
    - `slog.Debug()` - for debug messages
    - `slog.Warn()` - for warnings

## Common Replacements

### Basic prints
```go
// Before
fmt.Println("Starting all services...")

// After
slog.Info("Starting all services...")
```


### Formatted prints
```go
// Before
fmt.Printf("Starting service: %s...\n", service)

// After
slog.Info("Starting service", "service", service)
// or with slog.String for explicitness
slog.Info("Starting service", slog.String("service", service))
```


### Error output
```go
// Before
fmt.Fprintln(os.Stderr, err)

// After
slog.Error("Operation failed", "error", err)
// or
slog.Error("Operation failed", slog.Any("error", err))
```


## Setup (Optional but Recommended)

Configure the logger at the start of your `main()` function:

```go
func main() {
    // JSON handler (structured logs)
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    
    // Or text handler (human-readable)
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    
    // Set as default
    slog.SetDefault(logger)
    
    // Your existing code...
}
```


## Example Transformations for Your Code

**start.go:**
```go
// Before
fmt.Println("Starting all services...")
fmt.Printf("Starting service: %s...\n", service)

// After
slog.Info("Starting all services")
slog.Info("Starting service", "service", service)
```


**backup.go:**
```go
// Before
fmt.Println("Creating local backup... (stub)")

// After
slog.Info("Creating local backup", "status", "stub")
```


**main.go and root.go (errors):**
```go
// Before
fmt.Fprintln(os.Stderr, err)

// After
slog.Error("Command execution failed", "error", err)
```


## Benefits of `slog`

1. **Structured output**: Add key-value pairs for better log parsing
2. **Log levels**: Filter logs by severity
3. **Performance**: More efficient than string concatenation
4. **Context support**: Can pass context for request tracing
5. **Customizable**: Different handlers for different outputs (JSON, text, custom)

## Quick Migration Checklist

- [ ] Add `import "log/slog"` where needed
- [ ] Replace `fmt.Println()` → `slog.Info()`
- [ ] Replace `fmt.Printf()` → `slog.Info()` with key-value pairs
- [ ] Replace `fmt.Fprintln(os.Stderr, err)` → `slog.Error()`
- [ ] (Optional) Configure a custom handler in `main()`
- [ ] Remove unused `fmt` imports if no longer needed