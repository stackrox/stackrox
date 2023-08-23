# Go coding style guide

An overview of coding idioms and practiced style in stackrox/stackrox.

Whenever you encounter during code review (either as a reviewer or as a PR
author) something that you feel is non-obvious or not common sense and does not
fall into what can be considered common Go style, please add it to this guide.
Thanks!

## On consistency

There are often many ways to express the same thing (and not all of them are
correct). Consistency in code serves two main purposes:

- Avoiding errors of ignorance.
- Minimizing reader's surprise and hence facilitate reviews and maintenance.

When you deviate from a common pattern, consider leaving a comment explaining
why you do so. Prefer local consistency over global if you have to choose.

## On readability

One of the things that significantly impairs code readability is the amount of
non-local reasoning required to understand the contract and assumptions. Strive
to design your types and interfaces so that they can only be used in the
expected, correct way. 
```go
// Bad: do all `cache` users, current and future know how to use it correctly?  
var (
	cache = NewCache()
)
...
func (w *widget) Update(x X) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if cache.Contains(x) {
		...	
	}
	...
}
```

```go
// Better: `cache` users likely understand that manipulations require
// synchronization might miss future changes to this contract. 
var (
	cache = NewCache()
	mutex sync.RWMutex
)
...
func (w *widget) Update(x X) error {
	mutex.Lock()
	defer mutex.Unlock()

	if cache.Contains(x) {
		...	
	}
	...
}
```

```go
// Good: `cache` is only used by `widget` methods likely living in the same file.
type widget struct {
	cache Cache
	mutex sync.RWMutex
}
...
func (w *widget) Update(x X) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.cache.Contains(x) {
		...	
	}
	...
}
```

```go
// Good: code clearly relies on the fact that `cache` synchronizes inside.
type widget struct {
	cache sync.Cache // can be used concurrently 
}
...
func (w *widget) Update(x X) error {
	if w.cache.Contains(x) {
		...	
	}
	...
}
```

Capture in comments assumptions and decisions that are not obvious from the
code, for example:
```go
var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		// When this endpoint was public, Sensor relied on it to check Central's
		// availability. While Sensor might not do so today, we need to ensure
		// backward compatibility with older Sensors.
		or.SensorOr(user.Authenticated()): {
			"/v1.MetadataService/GetMetadata",
		},
	})
)
```

## Guidelines

- **Always** use `defer mutex.Unlock()` instead of explicitly calling `Unlock()`.
  If you need to unlock before the function returns, use `concurrency.WithLock()`
  or `concurrency.WithRLock()`.
- A `Close()` on a file etc. should only be deferred and wrapped in
  `utils.IgnoreError` if you exclusively _read_. If you write to a file, you
  **must** check for a `nil` error upon close, otherwise you cannot be certain
  that all data has been persisted.
- Use the fact that certain operations return zero values by default to write
  conditions in a more goal-oriented way. E.g., prefer
  `if x := m[k]; x != nil { foo(*x) }` over `if x, ok := m[k]; ok { foo(*x) }`.
  The former makes it apparent that the `*x` cannot panic while the latter only
  after checking that `m` indeed can never contain `nil` values.

### Declarations

- Prefer `var x T` instead of `x := T{}` or `x := 0/false/…` for initializing
  zero-valued local variables.
- Avoid `var ( … )` blocks within function scopes, only use them at the global
  scope.
- Declarations at the global scope should follow this order: const block, var
  block, everything else. If you define struct types and methods of this type in
  the same file, the order should be: type declaration, new/constructor function,
  methods. Do not interleave methods of different types.
- Right next to a `struct` definition, if applicable, declare which interfaces
  the struct implements: `var _ Interface = (*structImplementingInterface)(nil)`.
- If you are registering packages via `_` then extract this line into its own
  import block and preface it with a comment explaining what this import does.
- Scope variables as low as possible.
- Only use `if` + assignment with `:=`.

### Functions

- Every function that can potentially block should receive a `ctx context.Context`
  as the first parameter.
- An exported function should only ever return exported types (even though Go
  permits otherwise).
- Always pass around proto objects as pointers when traversing function
  boundaries. If you need to make a copy, use `obj.Clone()`.
- When using slice tricks like filtering in a function, pass the slice as a
  pointer to explicitly call out that the underlying data may be modified in
  that function.
- Avoid naked returns
```go
// Bad: naked return requires to look on the func declaration
// to know what is being returned. Moreover, there is a danger of
// shadowing a variable defined in the return arguments
// (e.g., `err`) later in the code.
func split(sum int) (x, y int) {
	x = sum * 4 / 9
	y = sum - x
	return
}
```

```go
// Good: it is clear what is returned by looking only at the return statement
func split(sum int) (x, y int) {
	x = sum * 4 / 9
	y = sum - x
	return x, y
}
```

### Types and collections

- Use `.GetField()` instead of `.Field` on protobuf objects
- Do not embed public types that are part of an object’s *internal* state into
  structs, exported or non-exported. I.e., do `mutex sync.Mutex` instead of
  `sync.Mutex` at the `struct` level. `obj.Lock()` suggests that Lock would be
  part of the public interface, which it should not be.
- When using `append`, preallocate whenever possible.
- When setting elements in a slice, instantiate like `s := make([]string, 0, capacity)`
  and use `append` because this greatly decreases potential index out of bounds
  errors.

### Error handling
 
- Use `errors.Wrap[f]()` from `github.com/pkg/errors` to add a message when
  forwarding the error.
- Use `RoxError.CausedBy[f]()` from `pkg/errox` to add context to an existing
  message. 
- Prefer `RoxError.New[f]()` from `pkg/errox` over `errors.Errorf()` from
  `github.com/pkg/errors` and `errors.New()` from the _builtin_ errors package
  to assign the error one of the standard classes.
- If you must define designated error conditions, do this in the package as
  global variables.
- When calling a function that returns an error, always check for `err != nil`
  before doing anything else with the results.
- When panic’ing, always use `error` objects as the argument.
- When doing a type conversion, use the form with two values on the left side of
  the assignment operator, even if you don’t use the value, e.g.,
  `x, _ := val.(T)`. The single-valued form may panic. If you know that the type
  conversion will always succeed, add a comment explaining why.

### Comments and text

- Use `TODO(ROX-XYZ)` for tracking what should be done in a follow-up.
- Aim for 120 columns max line length; consider 80 columns for comments and text
  blobs for readability.
