# GraphQL Backend documentation

The Stackrox GraphQL backend consists of several pieces.

Dependencies: we use the "graph-gophers" library to implement the
graphql protocol from here: https://github.com/graph-gophers/graphql-go.
That library requires us to wrap each type of object we expose via the
protocol in an object called a resolver.

The root resolver is found in `graphql/resolvers/resolver.go`. It has
references to datastores for every type of object we expose. *Support
for authorization will go there.*

`graphql/resolvers/generated.go` has generated code for resolvers for each
non-primitive type we expose. There are also a number of exposed utility
methods to convert the results of normal datastore lookups for every
wrapped type. The generated code assumes the root resolver type is called
Resolver, but otherwise takes no dependency on the rest of the codebase.

The rest of the files in `graphql/resolvers` are specific to each high
level type, and contain extra linking properties and definitions, as well
as defined entry points from the root resolver to each type. These are
exposed as methods on Resolver or the generated resolver types. This does
mean that the package will not compile without the generated code present.

`graphql/resolvers/gen/main.go` has the code generator main. It uses the
list of types from the schema package and invokes the go code generator
in `graphql/generator`. Because the `graphql/resolvers` package will not
compile without the generated code, this package cannot take a dependency
on it.

The `graphql/schema` package list of types we use as entry points to
generate both go code and the schema, as well as static utility methods
that the graphql/resolvers custom resolvers use to register themselves
with schema generation.

The `graphql/generator` package has a utility method to walk a set of
types, and the types of all of their fields, to generate a graphql schema
or generated resolver code. It also has the templates and utilities to
generate the text schema and the resolver code.

The `graphql/handler` package is the entry point for HTTP requests. It
ties together the root resolver and the text schema and passes them into
the graph-gophers engine.

# Code Scanning/Generation

The type scanner in `graphql/generator` takes a set of typed objects
(generally a set of nil pointers to proto message types will do fine, so
`((*v1.Type)(nil))`). It walks each field, looking for primitive types,
enums, other message pointers, and reads the proto message descriptor
to find oneof fields. It continues to iterate until all seen message
types have been exhausted.

The schema generator takes the output of that code walk, plus an
additional data map of extra resolver fields to append to existing
objects. The schema package has static methods, suitable for invoking
from package init() methods, that register these extra fields on the
root resolver, or on scanned types. Any additional fields registered
must be implemented manually in the resolver type.

The code generator takes the same list of output types and generates a
set of resolver types and methods. Each seen type has pointers to the
root resolver and the data object that it wraps, and standard getter
methods for each field.  There are also generated methods on the root
resolver to wrap individual messages and slices of messages.

If a type starts with the word "List" and has a matching type that doesn't
have the List prefix, it is subsumed into the resolver for the outer type,
and the resolver will automatically load the full object only if the
list object doesn't have the necessary data. Each outer type will need
a getFoo() method on the root resolver to enable loading the full object.

The schema generation uses the same logic and erases the List types. OneOf
types are mapped to graphql unions.

# Stability

If you add new fields to a proto struct, you will probably find a bunch
of tests failing. If so, run

    go generate central/graphql/resolvers

And commit the new generated.go file.
