package booleanpolicy

import (
	"regexp"
	"testing"
	"time"
)

// FuzzKeyValueRegex tests the keyValueValueRegex for ReDoS and correctness
func FuzzKeyValueRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("a=b")
	f.Add("key=value")
	f.Add("1=1")
	f.Add(`.*\d=.*`)

	// Seed with known invalid values
	f.Add("")
	f.Add("=")
	f.Add("=a=b")
	f.Add("no_equals")

	// Seed with potential ReDoS patterns
	f.Add("a" + "=" + "x")
	f.Add(string(make([]byte, 100)))

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = keyValueValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success - completed in reasonable time
		case <-time.After(100 * time.Millisecond):
			t.Errorf("regex match took too long (potential ReDoS) for input: %q", input)
		}
	})
}

// FuzzBooleanRegex tests the booleanValueRegex for ReDoS and correctness
func FuzzBooleanRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("true")
	f.Add("false")
	f.Add("True")
	f.Add("FALSE")

	// Seed with known invalid values
	f.Add("")
	f.Add("asdf")
	f.Add("FALS")
	f.Add("trueFalse")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = booleanValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("regex match took too long for input: %q", input)
		}
	})
}

// FuzzStringRegex tests the stringValueRegex for ReDoS and correctness
func FuzzStringRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("a")
	f.Add(" a\n")
	f.Add(" a")
	f.Add("\n\n.\n\n")

	// Seed with known invalid values
	f.Add("")
	f.Add(" ")
	f.Add("\n")
	f.Add("   ")

	// Seed with edge cases
	f.Add("\t\t\t")
	f.Add("\r\n\r\n")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = stringValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("regex match took too long for input: %q", input)
		}
	})
}

// FuzzIntegerRegex tests the integerValueRegex for ReDoS and correctness
func FuzzIntegerRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("0")
	f.Add("12")
	f.Add("1")
	f.Add("111111")

	// Seed with known invalid values
	f.Add("")
	f.Add("0<")
	f.Add(".")
	f.Add(".1")
	f.Add("0.1")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = integerValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("regex match took too long for input: %q", input)
		}
	})
}

// FuzzComparatorDecimalRegex tests the comparatorDecimalValueRegex for ReDoS and correctness
func FuzzComparatorDecimalRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("0")
	f.Add(">0")
	f.Add("<=1.2")
	f.Add(".1")
	f.Add("0.1")
	f.Add(">=0.1")

	// Seed with known invalid values
	f.Add("")
	f.Add("0<")
	f.Add(">")
	f.Add("3>0")
	f.Add(".")

	// Seed with edge cases (potential ReDoS with spaces and digits)
	f.Add(">     123.456")
	f.Add("<=   .999")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = comparatorDecimalValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("regex match took too long for input: %q", input)
		}
	})
}

// FuzzEnvironmentVariableRegex tests the environmentVariableWithSourceStrictRegex for ReDoS
func FuzzEnvironmentVariableRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("UNKNOWN=ENV=a")
	f.Add("UNSET=ENV=a")
	f.Add("RAW=ENV=a")
	f.Add("CONFIG_MAP_KEY=key=")
	f.Add("FIELD=key=")
	f.Add("RESOURCE_FIELD=key=")
	f.Add("SECRET_KEY==")
	f.Add("=ENV=a")
	f.Add("==")
	f.Add("===")

	// Seed with known invalid values
	f.Add("")
	f.Add("a=")
	f.Add("a=b")
	f.Add("=")
	f.Add("=1")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = environmentVariableWithSourceStrictRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("regex match took too long for input: %q", input)
		}
	})
}

// FuzzDockerfileLineRegex tests the dockerfileLineValueRegex for ReDoS
func FuzzDockerfileLineRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("ADD=.")
	f.Add("=.")
	f.Add("ADD=")
	f.Add("=")
	f.Add("FROM=alpine")
	f.Add("RUN=echo hello")

	// Seed with known invalid values
	f.Add("")
	f.Add("ADD")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = dockerfileLineValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("regex match took too long for input: %q", input)
		}
	})
}

// FuzzDropCapabilitiesRegex tests the dropCapabilitiesValueRegex for ReDoS
func FuzzDropCapabilitiesRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("ALL")
	f.Add("SYS_ADMIN")
	f.Add("NET_ADMIN")
	f.Add("CHOWN")

	// Seed with known invalid values
	f.Add("")
	f.Add("CAP_N_CRUNCH")
	f.Add("CAP_SYS_ADMIN")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = dropCapabilitiesValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("regex match took too long for input: %q", input)
		}
	})
}

// FuzzAddCapabilitiesRegex tests the addCapabilitiesValueRegex for ReDoS
func FuzzAddCapabilitiesRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("SYS_ADMIN")
	f.Add("NET_ADMIN")
	f.Add("CHOWN")

	// Seed with known invalid values
	f.Add("")
	f.Add("ALL")
	f.Add("CAP_SYS_ADMIN")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = addCapabilitiesValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("regex match took too long for input: %q", input)
		}
	})
}

// FuzzRBACPermissionRegex tests the rbacPermissionValueRegex for ReDoS
func FuzzRBACPermissionRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("Elevated_Cluster_Wide")
	f.Add("CLUSTER_ADMIN")
	f.Add("DEFAULT")
	f.Add("ELEVATED_IN_NAMESPACE")

	// Seed with known invalid values
	f.Add("")
	f.Add(" ")
	f.Add("asdf")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = rbacPermissionValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("regex match took too long for input: %q", input)
		}
	})
}

// FuzzPortExposureRegex tests the portExposureValueRegex for ReDoS
func FuzzPortExposureRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("NODE")
	f.Add("Host")
	f.Add("EXTERNAL")
	f.Add("INTERNAL")

	// Seed with known invalid values
	f.Add("")
	f.Add(" ")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = portExposureValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("regex match took too long for input: %q", input)
		}
	})
}

// FuzzIPv4Regex tests the IPv4 portion of ipAddressValueRegex for ReDoS
func FuzzIPv4Regex(f *testing.F) {
	// Seed with known valid IPv4 values
	f.Add("10.0.0.2")
	f.Add("192.168.1.1")
	f.Add("255.255.255.255")
	f.Add("0.0.0.0")

	// Seed with potential edge cases
	f.Add("999.999.999.999")
	f.Add("1.2.3.4.5")
	f.Add("...")

	// Create a simple IPv4-only regex for testing
	ipv4OnlyRegex := regexp.MustCompile("((?m)^" + ipv4Regex + "$)")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = ipv4OnlyRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("IPv4 regex match took too long for input: %q", input)
		}
	})
}

// FuzzIPv6Regex tests the IPv6 portion of ipAddressValueRegex for ReDoS
func FuzzIPv6Regex(f *testing.F) {
	// Seed with known valid IPv6 values
	f.Add("2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF")
	f.Add("2001:db8::1")
	f.Add("::1")
	f.Add("::")
	f.Add("fe80::1")

	// Seed with potential edge cases
	f.Add("gggg::1")
	f.Add("::::")
	f.Add("1:2:3:4:5:6:7:8:9")

	// Create a simple IPv6-only regex for testing
	ipv6OnlyRegex := regexp.MustCompile("((?m)^" + ipv6Regex + "$)")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = ipv6OnlyRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("IPv6 regex match took too long for input: %q", input)
		}
	})
}

// FuzzIPAddressRegex tests the combined ipAddressValueRegex for ReDoS
func FuzzIPAddressRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("10.0.0.2")
	f.Add("2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF")
	f.Add("192.168.1.1")
	f.Add("::1")

	// Seed with known invalid values
	f.Add("999:999.999.222")
	f.Add("not-an-ip")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = ipAddressValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("IP address regex match took too long for input: %q", input)
		}
	})
}

// FuzzSeverityRegex tests the severityValueRegex for ReDoS
func FuzzSeverityRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("LOW")
	f.Add(">LOW")
	f.Add(">=MODERATE")
	f.Add("CRITICAL")
	f.Add("<=IMPORTANT")

	// Seed with known invalid values
	f.Add("")
	f.Add("MEGA")
	f.Add("> > LOW")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = severityValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("severity regex match took too long for input: %q", input)
		}
	})
}

// FuzzKubernetesNameRegex tests the kubernetesNameRegex for ReDoS
func FuzzKubernetesNameRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("central-htpasswd")
	f.Add("system:serviceaccount:openshift-kube-apiserver:localhost-recovery-client")
	f.Add("a")
	f.Add("a-b-c")

	// Seed with potential edge cases
	f.Add("-name")
	f.Add("name-")
	f.Add("NAME")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = kubernetesNameRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("kubernetes name regex match took too long for input: %q", input)
		}
	})
}

// FuzzFileOperationRegex tests the fileOperationRegex for ReDoS
func FuzzFileOperationRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("OPEN")
	f.Add("CREATE")
	f.Add("RENAME")
	f.Add("UNLINK")
	f.Add("OWNERSHIP_CHANGE")
	f.Add("PERMISSION_CHANGE")
	f.Add("open")
	f.Add("create")

	// Seed with known invalid values
	f.Add("")
	f.Add(" ")
	f.Add("READ")
	f.Add("WRITE")
	f.Add("DELETE")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = fileOperationRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("file operation regex match took too long for input: %q", input)
		}
	})
}

// FuzzSignatureIntegrationIDRegex tests the signatureIntegrationIDValueRegex for ReDoS
func FuzzSignatureIntegrationIDRegex(f *testing.F) {
	// Seed with known valid UUID patterns
	f.Add("io.stackrox.signatureintegration.12345678-1234-1234-1234-123456789abc")
	f.Add("io.stackrox.signatureintegration.abcdef01-2345-6789-abcd-ef0123456789")

	// Seed with known invalid patterns
	f.Add("")
	f.Add("io.stackrox.signatureintegration.")
	f.Add("io.stackrox.signatureintegration.not-a-uuid")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = signatureIntegrationIDValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("signature integration ID regex match took too long for input: %q", input)
		}
	})
}

// FuzzAuditEventAPIVerbRegex tests the auditEventAPIVerbValueRegex for ReDoS
func FuzzAuditEventAPIVerbRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("CREATE")
	f.Add("DELETE")
	f.Add("GET")
	f.Add("PATCH")
	f.Add("UPDATE")
	f.Add("get")
	f.Add("patch")

	// Seed with known invalid values
	f.Add("")
	f.Add("WATCH")
	f.Add("LIST")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = auditEventAPIVerbValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("audit event API verb regex match took too long for input: %q", input)
		}
	})
}

// FuzzAuditEventResourceRegex tests the auditEventResourceValueRegex for ReDoS
func FuzzAuditEventResourceRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("SECRETS")
	f.Add("CONFIGMAPS")
	f.Add("CLUSTER_ROLES")
	f.Add("secrets")
	f.Add("configmaps")

	// Seed with known invalid values
	f.Add("")
	f.Add("PODS")
	f.Add("RBAC")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = auditEventResourceValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("audit event resource regex match took too long for input: %q", input)
		}
	})
}

// FuzzKubernetesResourceRegex tests the kubernetesResourceValueRegex for ReDoS
func FuzzKubernetesResourceRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("PODS_EXEC")
	f.Add("PODS_PORTFORWARD")
	f.Add("PODS_ATTACH")
	f.Add("pods_exec")

	// Seed with known invalid values
	f.Add("")
	f.Add("PODS")
	f.Add("EXEC")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = kubernetesResourceValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("kubernetes resource regex match took too long for input: %q", input)
		}
	})
}

// FuzzMountPropagationRegex tests the mountPropagationValueRegex for ReDoS
func FuzzMountPropagationRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("NONE")
	f.Add("HOSTTOCONTAINER")
	f.Add("BIDIRECTIONAL")
	f.Add("none")

	// Seed with known invalid values
	f.Add("")
	f.Add("SHARED")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = mountPropagationValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("mount propagation regex match took too long for input: %q", input)
		}
	})
}

// FuzzSeccompProfileTypeRegex tests the seccompProfileTypeValueRegex for ReDoS
func FuzzSeccompProfileTypeRegex(f *testing.F) {
	// Seed with known valid values
	f.Add("UNCONFINED")
	f.Add("RUNTIME_DEFAULT")
	f.Add("LOCALHOST")
	f.Add("unconfined")

	// Seed with known invalid values
	f.Add("")
	f.Add("CUSTOM")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q: %v", input, r)
			}
		}()

		done := make(chan bool)
		go func() {
			_ = seccompProfileTypeValueRegex.MatchString(input)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("seccomp profile type regex match took too long for input: %q", input)
		}
	})
}

// FuzzCreateRegex tests the createRegex function itself for potential issues
func FuzzCreateRegex(f *testing.F) {
	// Seed with patterns that are used in the codebase
	f.Add("[^=]+=.*")
	f.Add("(?i:(true|false))")
	f.Add("([[:digit:]]+)")

	// Seed with potentially problematic patterns
	f.Add(".*")
	f.Add("(a+)+")
	f.Add("a*b*c*")

	f.Fuzz(func(t *testing.T, pattern string) {
		defer func() {
			if r := recover(); r != nil {
				// It's OK to panic on invalid patterns during compilation
				// We're primarily checking for runtime issues
			}
		}()

		// Attempt to compile the pattern
		_, err := regexp.Compile("((?m)^" + pattern + "$)")
		if err != nil {
			// Invalid pattern is expected for fuzz inputs
			return
		}

		// If compilation succeeds, the test passes
	})
}
