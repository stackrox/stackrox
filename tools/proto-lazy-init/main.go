// proto-lazy-init transforms protoc-gen-go output to use lazy initialization.
// Instead of registering proto types in init(), registration is deferred to
// the first ProtoReflect() call via sync.Once.
//
// This means binaries that never call ProtoReflect() (sensor, admission-control)
// save ~10-15 MB of heap. Binaries that do (central, roxctl) pay the cost on
// first use — typically during startup when grpc-gateway initializes.
//
// Usage: proto-lazy-init <file.pb.go>
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	// Matches: func init() { file_storage_alert_proto_init() }
	initRe = regexp.MustCompile(`^func init\(\) \{ (file_\w+_proto_init)\(\) \}$`)
	// Matches: func (x *Alert) ProtoReflect() protoreflect.Message {
	protoReflectRe = regexp.MustCompile(`^func \(x \*\w+\) ProtoReflect\(\) protoreflect\.Message \{`)
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: proto-lazy-init <file.pb.go>\n")
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// First pass: find the init function name (it's at the end of the file)
	var initFuncName string
	for _, line := range lines {
		if m := initRe.FindStringSubmatch(line); m != nil {
			initFuncName = m[1]
			break
		}
	}
	if initFuncName == "" {
		return // no proto init() in this file
	}

	var out []string
	addedSync := false

	for _, line := range lines {
		// Transform the init() line
		if m := initRe.FindStringSubmatch(line); m != nil {
			// Replace eager init() with sync.Once lazy trigger
			out = append(out,
				fmt.Sprintf("var %s_once sync.Once", initFuncName),
				fmt.Sprintf("func %s_ensure() { %s_once.Do(%s) }", initFuncName, initFuncName, initFuncName),
				"func init() {} // proto registration is lazy — triggered by first ProtoReflect() call",
			)
			continue
		}

		// Add ensure() call at the top of every ProtoReflect method
		if strings.Contains(line, "ProtoReflect()") && strings.Contains(line, "func (x *") {
			out = append(out, line)
			out = append(out, fmt.Sprintf("\t%s_ensure()", initFuncName))
			continue
		}

		// Add ensure() call at the top of every enum Descriptor() method
		// Enum String() calls EnumStringOf(x.Descriptor(),...) which needs the registry.
		if strings.Contains(line, ".Descriptor()") && strings.Contains(line, "func (") && strings.Contains(line, ") Descriptor() protoreflect.EnumDescriptor {") {
			out = append(out, line)
			out = append(out, fmt.Sprintf("\t%s_ensure()", initFuncName))
			continue
		}

		// Add "sync" to imports if needed (only if not already imported)
		if !addedSync && strings.Contains(line, `protoimpl "google.golang.org/protobuf/runtime/protoimpl"`) {
			if !strings.Contains(content, "\"sync\"") {
				out = append(out, "\t\"sync\"")
			}
			addedSync = true
		}

		out = append(out, line)
	}

	if err := os.WriteFile(os.Args[1], []byte(strings.Join(out, "\n")), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
