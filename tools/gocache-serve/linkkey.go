package main

// Vendored replication of cmd/go's linkActionID
// (cmd/go/internal/work/exec.go, Go 1.27.1). Computed pre-build to
// predict a link's actionID — the key for link-output caching. The
// toolchain's exact recipe was captured via GODEBUG=gocachehash=1 and
// is re-verified against the real toolchain by TestLinkKeyMatchesToolchain:
// if go changes its key generation, that test fails loudly and this
// file needs updating (toolchain pinned by go.mod bounds the drift
// window; the binary's buildID check still validates every skip).
//
// Hash structure: sha256(salt || sequence), salt = the go version string
// (cache.NewHash). The actionID used in buildIDs is base64(sum[:15]).

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// LinkKeyDep is one entry of the link's dependency list.
type LinkKeyDep struct {
	ImportPath string
	ContentID  string // the contentID half of the dep's buildID
	IsMain     bool   // main package: emits the packagemain line after its packagefile
}

// LinkKeyInput carries the structured inputs to a link action's ID.
type LinkKeyInput struct {
	GoVersion  string // e.g. "go1.27.1" (the hash salt)
	ImportPath string

	Buildmode, Goos, Goarch    string // BuildBuildmode default renders "default"
	OmitDebug, Standard, Local bool
	LocalPrefix                string
	DefaultGODEBUG             string

	Trimpath bool

	ToolID         string   // `go tool link -V=full` first field set, e.g. "link version go1.27.1"
	ForcedLdflags  []string // global forced ldflags
	LdBuildmode    string   // e.g. "exe"
	PackageLdflags []string // the -X x_defs etc. for this package

	ArchEnvKey, ArchEnvVal string // GOAMD64=v1 class
	GOEXPERIMENT           string
	Goroot                 string // empty when trimpath
	GoExtlinkEnabled       string

	Deps        []LinkKeyDep // in the link action's dependency order
	MainBuildID string       // full actionID/contentID of main.a
}

// RenderBytes produces the exact byte sequence hashed, for debugging
// and test comparison against GODEBUG=gocachehash=1 dumps.
func (in *LinkKeyInput) RenderBytes() []byte {
	h := &bytes.Buffer{}
	fmt.Fprintf(h, "link\n")
	fmt.Fprintf(h, "buildmode %s goos %s goarch %s\n", in.Buildmode, in.Goos, in.Goarch)
	fmt.Fprintf(h, "import %q\n", in.ImportPath)
	fmt.Fprintf(h, "omitdebug %v standard %v local %v prefix %q\n", in.OmitDebug, in.Standard, in.Local, in.LocalPrefix)
	fmt.Fprintf(h, "defaultgodebug %q\n", in.DefaultGODEBUG)
	if in.Trimpath {
		fmt.Fprintln(h, "trimpath")
	}
	fmt.Fprintf(h, "link %s %q %s\n", in.ToolID, in.ForcedLdflags, in.LdBuildmode)
	fmt.Fprintf(h, "linkflags %q\n", in.PackageLdflags)
	fmt.Fprintf(h, "%s=%s\n", in.ArchEnvKey, in.ArchEnvVal)
	if in.GOEXPERIMENT != "" {
		fmt.Fprintf(h, "GOEXPERIMENT=%q\n", in.GOEXPERIMENT)
	}
	fmt.Fprintf(h, "GOROOT=%s\n", in.Goroot)
	fmt.Fprintf(h, "GO_EXTLINK_ENABLED=%s\n", in.GoExtlinkEnabled)
	for _, d := range in.Deps {
		fmt.Fprintf(h, "packagefile %s=%s\n", d.ImportPath, d.ContentID)
		if d.IsMain && in.MainBuildID != "" {
			// cmd/go emits packagemain inside the dep loop, immediately
			// after the main package's packagefile line.
			fmt.Fprintf(h, "packagemain %s\n", in.MainBuildID)
		}
	}
	return h.Bytes()
}

// Hash renders the exact byte sequence cmd/go hashes and returns the
// sha256 sum.
func (in *LinkKeyInput) Hash() [32]byte {
	h := sha256.New()
	h.Write([]byte(in.GoVersion))
	h.Write(in.RenderBytes())
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// ActionID renders the buildID actionID form: URL-safe base64 (no
// padding) of the first 15 bytes (buildid.HashToString — RawURLEncoding,
// which the multi-arch test catches: standard encoding differs on the
// +/ vs -_ alphabet).
func LinkKeyActionID(sum [32]byte) string {
	return base64.RawURLEncoding.EncodeToString(sum[:15])
}
