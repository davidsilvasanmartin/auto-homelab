# `t.Fatal` vs `t.Error`: When to Use Each

## The Key Difference

- **`t.Error()`**: Reports the failure but **continues executing the test**
- **`t.Fatal()`**: Reports the failure and **immediately stops the test**

## Best Practice Rule (Simple Version)

**Use `t.Fatal()` when continuing the test would cause a panic or produce meaningless/misleading results.**

## Decision Guide

```go
// ❌ BAD: Will panic on next line if err is not nil
if err != nil {
    t.Error("expected no error")
}
result.DoSomething() // PANIC if result is nil!

// ✅ GOOD: Stop immediately to avoid panic
if err != nil {
    t.Fatal("expected no error")
}
result.DoSomething() // Safe - won't reach here if err != nil
```


## When to Use `t.Fatal()`

Use `t.Fatal()` when the failure is a **prerequisite** for the rest of the test:

1. **Nil checks before dereferencing**
2. **Error checks when you need to use the result**
3. **Setup failures** (can't proceed without proper setup)
4. **Assertion failures that make subsequent checks meaningless**

```go
// ✅ Use Fatal - need the value to continue
value, err := env.GetRequiredEnv("API_KEY")
if err != nil {
    t.Fatal("expected no error, got:", err) // STOP - can't test value
}
// Now we can safely use value
if len(value) == 0 {
    t.Error("expected non-empty value")
}

// ✅ Use Fatal - need non-nil to continue
var customErr *CustomError
if !errors.As(err, &customErr) {
    t.Fatal("expected CustomError") // STOP - can't access fields
}
// Now we can safely access customErr.Field
if customErr.Count != 2 {
    t.Error("wrong count")
}
```


## When to Use `t.Error()`

Use `t.Error()` when you want to **collect multiple failures** in one test run:

1. **Independent assertions**
2. **Multiple field validations**
3. **When you want to see all failures at once**

```go
// ✅ Use Error - independent checks, collect all failures
if capturedName != "docker" {
    t.Error("wrong command name") // Continue to check args too
}
if len(capturedArgs) != 3 {
    t.Error("wrong arg count") // Continue to check specific args
}
if capturedArgs[0] != "compose" {
    t.Error("wrong first arg") // See all failures in one test run
}
```


## Fixing Your Tests

Here's how to apply these rules to your code:

### Pattern 1: Error Check Where You Need the Result

```go
func TestDefaultEnv_GetRequiredEnv_Success(t *testing.T) {
	value := "localhost:5432/db"
	mock := &mockEnvLookup{
		lookupEnvFunc: func(key string) (string, bool) {
			return value, true
		},
	}
	env := &DefaultEnv{LookupEnv: mock.lookupEnv}

	obtainedValue, err := env.GetRequiredEnv("DB_URL")

	// ✅ Use Fatal - we need obtainedValue to be valid for next check
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// ✅ Use Error - this is independent, just reporting a mismatch
	if obtainedValue != value {
		t.Errorf("expected value %q, got %q", value, obtainedValue)
	}
}
```


### Pattern 2: Nil Check Before Further Assertions

Your tests are already following this correctly! Example:

```go
func TestDefaultFilesHandler_CreateDirIfNotExists_PathNotAbs(t *testing.T) {
	files := &DefaultFilesHandler{stdlib: &mockStdlib{}}

	err := files.CreateDirIfNotExists("relative/path")

	// ✅ Correct use of Fatal - need err to be non-nil for next check
	if err == nil {
		t.Fatal("expected error when path is relative, got nil")
	}
	// ✅ Use Error - just checking error type
	if !errors.Is(err, ErrPathNotAbsolute) {
		t.Errorf("expected ErrPathNotAbsolute, got: %v", err)
	}
}
```


### Pattern 3: Multiple Independent Checks

Your runner tests are already excellent! They use `Fatal` for the first error check (since you need the command to have run), then `Error` for independent validation:

```go
func TestSystemRunner_ComposeStart_NoServices(t *testing.T) {
	// ... setup ...

	err := runner.ComposeStart([]string{})

	// ✅ Use Fatal - if there's an error, the rest of the test is meaningless
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// ✅ Use Error - independent checks, want to see all failures
	if capturedName != "docker" {
		t.Errorf("expected command name %q, got %q", "docker", capturedName)
	}
	expectedArgs := []string{"compose", "up", "-d"}
	if diff := cmp.Diff(expectedArgs, capturedArgs); diff != "" {
		t.Errorf("args mismatch:\n%s", diff)
	}
}
```


## Common Patterns Summary

```go
// Pattern 1: Need the result value
result, err := function()
if err != nil {
    t.Fatal() // ✅ Stop - can't use result
}
// Use result safely here

// Pattern 2: Expect an error, need to inspect it
err := function()
if err == nil {
    t.Fatal() // ✅ Stop - can't inspect nil error
}
// Check error type/contents safely here

// Pattern 3: Multiple independent assertions
if condition1 {
    t.Error() // ✅ Continue to see all failures
}
if condition2 {
    t.Error() // ✅ Continue to see all failures
}
if condition3 {
    t.Error() // ✅ Continue to see all failures
}

// Pattern 4: Success case with multiple validations
if result == nil {
    t.Fatal() // ✅ Stop - can't dereference nil
}
// All Error() below - independent field checks
if result.Field1 != expected1 {
    t.Error()
}
if result.Field2 != expected2 {
    t.Error()
}
```


## Your Tests Status: Already Pretty Good! ✅

Looking at your code, you're **already following best practices** in most places:

- ✅ Using `Fatal` when checking for nil errors before using results
- ✅ Using `Error` for independent field validations
- ✅ Using `Fatal` when expecting an error before inspecting it

The only small improvement would be ensuring consistency. Your current pattern is solid!

## Quick Reference

| Situation | Use |
|-----------|-----|
| Check error before using result | `t.Fatal()` |
| Check result is not nil before dereferencing | `t.Fatal()` |
| Multiple independent field validations | `t.Error()` |
| Want to see all failures at once | `t.Error()` |
| Setup/prerequisite failure | `t.Fatal()` |
| Isolated assertion that stands alone | `t.Error()` |

**Golden Rule**: If the next line would panic or be meaningless without the check passing, use `t.Fatal()`. Otherwise, use `t.Error()`.