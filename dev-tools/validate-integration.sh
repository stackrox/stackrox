#!/usr/bin/env bash
set -euo pipefail

# Validate local-dev flavor integration with Helm templates
# This script tests the complete integration without actually building images

echo "🧪 Validating local-dev flavor integration..."

# Test 1: Verify local-dev flavor exists and works
echo ""
echo "Test 1: Local-dev flavor availability"

cat > /tmp/test_flavor_exists.go << 'EOF'
package main

import (
    "fmt"
    "github.com/stackrox/rox/pkg/images/defaults"
)

func main() {
    _, err := defaults.GetImageFlavorByName("local-dev", false)
    if err != nil {
        panic(err)
    }
    fmt.Println("✅ local-dev flavor available")
}
EOF

if go run /tmp/test_flavor_exists.go; then
    echo "✅ local-dev flavor is available"
else
    echo "❌ local-dev flavor failed to load"
    exit 1
fi

# Test 2: Environment variable configuration
echo ""
echo "Test 2: Environment variable configuration"
export ROX_LOCAL_REGISTRY=test-registry:5000
export ROX_LOCAL_TAG=test-tag

# Create a simple test program
cat > /tmp/test_flavor.go << 'EOF'
package main

import (
    "fmt"
    "github.com/stackrox/rox/pkg/images/defaults"
)

func main() {
    flavor, err := defaults.GetImageFlavorByName("local-dev", false)
    if err != nil {
        panic(err)
    }

    expectedRegistry := "test-registry:5000"
    expectedTag := "test-tag"

    if flavor.MainRegistry != expectedRegistry {
        panic(fmt.Sprintf("Expected registry %s, got %s", expectedRegistry, flavor.MainRegistry))
    }

    if flavor.MainImageTag != expectedTag {
        panic(fmt.Sprintf("Expected tag %s, got %s", expectedTag, flavor.MainImageTag))
    }

    fmt.Printf("✅ Registry: %s\n", flavor.MainRegistry)
    fmt.Printf("✅ Tag: %s\n", flavor.MainImageTag)
    fmt.Printf("✅ Main image: %s\n", flavor.MainImage())
    fmt.Printf("✅ Central DB image: %s\n", flavor.CentralDBImage())
    fmt.Printf("✅ Collector image: %s\n", flavor.CollectorImage())
}
EOF

if go run /tmp/test_flavor.go; then
    echo "✅ Environment variable configuration works"
else
    echo "❌ Environment variable configuration failed"
    exit 1
fi

# Test 3: Flavor selection via environment
echo ""
echo "Test 3: Flavor selection via ROX_IMAGE_FLAVOR"
export ROX_IMAGE_FLAVOR=local-dev

cat > /tmp/test_env_flavor.go << 'EOF'
package main

import (
    "fmt"
    "github.com/stackrox/rox/pkg/images/defaults"
)

func main() {
    flavor := defaults.GetImageFlavorFromEnv()

    if flavor.MainRegistry != "test-registry:5000" {
        panic("ROX_IMAGE_FLAVOR=local-dev not working")
    }

    fmt.Println("✅ ROX_IMAGE_FLAVOR=local-dev works correctly")
}
EOF

if go run /tmp/test_env_flavor.go; then
    echo "✅ Environment flavor selection works"
else
    echo "❌ Environment flavor selection failed"
    exit 1
fi

# Test 4: Default values
echo ""
echo "Test 4: Default values without environment variables"
unset ROX_LOCAL_REGISTRY ROX_LOCAL_TAG

cat > /tmp/test_defaults.go << 'EOF'
package main

import (
    "fmt"
    "github.com/stackrox/rox/pkg/images/defaults"
)

func main() {
    flavor, err := defaults.GetImageFlavorByName("local-dev", false)
    if err != nil {
        panic(err)
    }

    if flavor.MainRegistry != "localhost:5000" {
        panic(fmt.Sprintf("Expected default registry localhost:5000, got %s", flavor.MainRegistry))
    }

    if flavor.MainImageTag != "latest" {
        panic(fmt.Sprintf("Expected default tag latest, got %s", flavor.MainImageTag))
    }

    fmt.Printf("✅ Default registry: %s\n", flavor.MainRegistry)
    fmt.Printf("✅ Default tag: %s\n", flavor.MainImageTag)
}
EOF

if go run /tmp/test_defaults.go; then
    echo "✅ Default values work correctly"
else
    echo "❌ Default values failed"
    exit 1
fi

# Test 5: Validate Tekton resources syntax
echo ""
echo "Test 5: Tekton resource validation"
for file in dev-tools/tekton/*.yaml; do
    # Skip PipelineRun with generateName (can't be validated with apply)
    if [[ "$(basename "$file")" == "pipelinerun-local-dev.yaml" ]]; then
        if kubectl --dry-run=client create -f "$file" >/dev/null 2>&1; then
            echo "✅ $(basename "$file") syntax valid"
        else
            echo "❌ $(basename "$file") syntax invalid"
            exit 1
        fi
    elif kubectl --dry-run=client apply -f "$file" >/dev/null 2>&1; then
        echo "✅ $(basename "$file") syntax valid"
    else
        echo "❌ $(basename "$file") syntax invalid"
        exit 1
    fi
done

# Test 6: Wrapper script help
echo ""
echo "Test 6: Wrapper script functionality"
if ./dev-tools/local-build.sh --help >/dev/null 2>&1; then
    echo "✅ Wrapper script help works"
else
    echo "❌ Wrapper script help failed"
    exit 1
fi

# Cleanup
rm -f /tmp/test_flavor_exists.go /tmp/test_flavor.go /tmp/test_env_flavor.go /tmp/test_defaults.go

echo ""
echo "🎉 All validation tests passed!"
echo ""
echo "The fast local Tekton build system is ready for use:"
echo ""
echo "1. Build images:     ./dev-tools/local-build.sh"
echo "2. Custom config:    ./dev-tools/local-build.sh --registry my-reg:5000 --tag v1.0.0"
echo "3. Build and deploy: ./dev-tools/local-build.sh --deploy"
echo "4. Use custom images: export ROX_IMAGE_FLAVOR=local-dev && ./installer/bin/installer apply central"
echo ""
echo "See dev-tools/README-local-dev.md for complete documentation."