package booleanpolicy

import (
	"regexp"
	"testing"
)

func FuzzKeyValueRegex(f *testing.F) {
	f.Add("a=b")
	f.Add("key=value")
	f.Add("1=1")
	f.Add(`.*\d=.*`)
	f.Add("")
	f.Add("=")
	f.Add("=a=b")
	f.Add("no_equals")
	f.Add("a" + "=" + "x")
	f.Add(string(make([]byte, 100)))

	f.Fuzz(func(_ *testing.T, input string) {
		_ = keyValueValueRegex.MatchString(input)
	})
}

func FuzzBooleanRegex(f *testing.F) {
	f.Add("true")
	f.Add("false")
	f.Add("True")
	f.Add("FALSE")
	f.Add("")
	f.Add("asdf")
	f.Add("FALS")
	f.Add("trueFalse")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = booleanValueRegex.MatchString(input)
	})
}

func FuzzStringRegex(f *testing.F) {
	f.Add("a")
	f.Add(" a\n")
	f.Add(" a")
	f.Add("\n\n.\n\n")
	f.Add("")
	f.Add(" ")
	f.Add("\n")
	f.Add("   ")
	f.Add("\t\t\t")
	f.Add("\r\n\r\n")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = stringValueRegex.MatchString(input)
	})
}

func FuzzIntegerRegex(f *testing.F) {
	f.Add("0")
	f.Add("12")
	f.Add("1")
	f.Add("111111")
	f.Add("")
	f.Add("0<")
	f.Add(".")
	f.Add(".1")
	f.Add("0.1")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = integerValueRegex.MatchString(input)
	})
}

func FuzzComparatorDecimalRegex(f *testing.F) {
	f.Add("0")
	f.Add(">0")
	f.Add("<=1.2")
	f.Add(".1")
	f.Add("0.1")
	f.Add(">=0.1")
	f.Add("")
	f.Add("0<")
	f.Add(">")
	f.Add("3>0")
	f.Add(".")
	f.Add(">     123.456")
	f.Add("<=   .999")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = comparatorDecimalValueRegex.MatchString(input)
	})
}

func FuzzEnvironmentVariableRegex(f *testing.F) {
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
	f.Add("")
	f.Add("a=")
	f.Add("a=b")
	f.Add("=")
	f.Add("=1")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = environmentVariableWithSourceStrictRegex.MatchString(input)
	})
}

func FuzzDockerfileLineRegex(f *testing.F) {
	f.Add("ADD=.")
	f.Add("=.")
	f.Add("ADD=")
	f.Add("=")
	f.Add("FROM=alpine")
	f.Add("RUN=echo hello")
	f.Add("")
	f.Add("ADD")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = dockerfileLineValueRegex.MatchString(input)
	})
}

func FuzzDropCapabilitiesRegex(f *testing.F) {
	f.Add("ALL")
	f.Add("SYS_ADMIN")
	f.Add("NET_ADMIN")
	f.Add("CHOWN")
	f.Add("")
	f.Add("CAP_N_CRUNCH")
	f.Add("CAP_SYS_ADMIN")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = dropCapabilitiesValueRegex.MatchString(input)
	})
}

func FuzzAddCapabilitiesRegex(f *testing.F) {
	f.Add("SYS_ADMIN")
	f.Add("NET_ADMIN")
	f.Add("CHOWN")
	f.Add("")
	f.Add("ALL")
	f.Add("CAP_SYS_ADMIN")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = addCapabilitiesValueRegex.MatchString(input)
	})
}

func FuzzRBACPermissionRegex(f *testing.F) {
	f.Add("Elevated_Cluster_Wide")
	f.Add("CLUSTER_ADMIN")
	f.Add("DEFAULT")
	f.Add("ELEVATED_IN_NAMESPACE")
	f.Add("")
	f.Add(" ")
	f.Add("asdf")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = rbacPermissionValueRegex.MatchString(input)
	})
}

func FuzzPortExposureRegex(f *testing.F) {
	f.Add("NODE")
	f.Add("Host")
	f.Add("EXTERNAL")
	f.Add("INTERNAL")
	f.Add("")
	f.Add(" ")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = portExposureValueRegex.MatchString(input)
	})
}

func FuzzIPv4Regex(f *testing.F) {
	f.Add("10.0.0.2")
	f.Add("192.168.1.1")
	f.Add("255.255.255.255")
	f.Add("0.0.0.0")
	f.Add("999.999.999.999")
	f.Add("1.2.3.4.5")
	f.Add("...")

	ipv4OnlyRegex := regexp.MustCompile("((?m)^" + ipv4Regex + "$)")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = ipv4OnlyRegex.MatchString(input)
	})
}

func FuzzIPv6Regex(f *testing.F) {
	f.Add("2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF")
	f.Add("2001:db8::1")
	f.Add("::1")
	f.Add("::")
	f.Add("fe80::1")
	f.Add("gggg::1")
	f.Add("::::")
	f.Add("1:2:3:4:5:6:7:8:9")

	ipv6OnlyRegex := regexp.MustCompile("((?m)^" + ipv6Regex + "$)")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = ipv6OnlyRegex.MatchString(input)
	})
}

func FuzzIPAddressRegex(f *testing.F) {
	f.Add("10.0.0.2")
	f.Add("2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF")
	f.Add("192.168.1.1")
	f.Add("::1")
	f.Add("999:999.999.222")
	f.Add("not-an-ip")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = ipAddressValueRegex.MatchString(input)
	})
}

func FuzzSeverityRegex(f *testing.F) {
	f.Add("LOW")
	f.Add(">LOW")
	f.Add(">=MODERATE")
	f.Add("CRITICAL")
	f.Add("<=IMPORTANT")
	f.Add("")
	f.Add("MEGA")
	f.Add("> > LOW")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = severityValueRegex.MatchString(input)
	})
}

func FuzzKubernetesNameRegex(f *testing.F) {
	f.Add("central-htpasswd")
	f.Add("system:serviceaccount:openshift-kube-apiserver:localhost-recovery-client")
	f.Add("a")
	f.Add("a-b-c")
	f.Add("-name")
	f.Add("name-")
	f.Add("NAME")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = kubernetesNameRegex.MatchString(input)
	})
}

func FuzzFileOperationRegex(f *testing.F) {
	f.Add("OPEN")
	f.Add("CREATE")
	f.Add("RENAME")
	f.Add("UNLINK")
	f.Add("OWNERSHIP_CHANGE")
	f.Add("PERMISSION_CHANGE")
	f.Add("open")
	f.Add("create")
	f.Add("")
	f.Add(" ")
	f.Add("READ")
	f.Add("WRITE")
	f.Add("DELETE")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = fileOperationRegex.MatchString(input)
	})
}

func FuzzSignatureIntegrationIDRegex(f *testing.F) {
	f.Add("io.stackrox.signatureintegration.12345678-1234-1234-1234-123456789abc")
	f.Add("io.stackrox.signatureintegration.abcdef01-2345-6789-abcd-ef0123456789")
	f.Add("")
	f.Add("io.stackrox.signatureintegration.")
	f.Add("io.stackrox.signatureintegration.not-a-uuid")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = signatureIntegrationIDValueRegex.MatchString(input)
	})
}

func FuzzAuditEventAPIVerbRegex(f *testing.F) {
	f.Add("CREATE")
	f.Add("DELETE")
	f.Add("GET")
	f.Add("PATCH")
	f.Add("UPDATE")
	f.Add("get")
	f.Add("patch")
	f.Add("")
	f.Add("WATCH")
	f.Add("LIST")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = auditEventAPIVerbValueRegex.MatchString(input)
	})
}

func FuzzAuditEventResourceRegex(f *testing.F) {
	f.Add("SECRETS")
	f.Add("CONFIGMAPS")
	f.Add("CLUSTER_ROLES")
	f.Add("secrets")
	f.Add("configmaps")
	f.Add("")
	f.Add("PODS")
	f.Add("RBAC")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = auditEventResourceValueRegex.MatchString(input)
	})
}

func FuzzKubernetesResourceRegex(f *testing.F) {
	f.Add("PODS_EXEC")
	f.Add("PODS_PORTFORWARD")
	f.Add("PODS_ATTACH")
	f.Add("pods_exec")
	f.Add("")
	f.Add("PODS")
	f.Add("EXEC")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = kubernetesResourceValueRegex.MatchString(input)
	})
}

func FuzzMountPropagationRegex(f *testing.F) {
	f.Add("NONE")
	f.Add("HOSTTOCONTAINER")
	f.Add("BIDIRECTIONAL")
	f.Add("none")
	f.Add("")
	f.Add("SHARED")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = mountPropagationValueRegex.MatchString(input)
	})
}

func FuzzSeccompProfileTypeRegex(f *testing.F) {
	f.Add("UNCONFINED")
	f.Add("RUNTIME_DEFAULT")
	f.Add("LOCALHOST")
	f.Add("unconfined")
	f.Add("")
	f.Add("CUSTOM")

	f.Fuzz(func(_ *testing.T, input string) {
		_ = seccompProfileTypeValueRegex.MatchString(input)
	})
}

func FuzzCreateRegex(f *testing.F) {
	f.Add("[^=]+=.*")
	f.Add("(?i:(true|false))")
	f.Add("([[:digit:]]+)")
	f.Add(".*")
	f.Add("(a+)+")
	f.Add("a*b*c*")

	f.Fuzz(func(_ *testing.T, pattern string) {
		// Attempt to compile the pattern - panics on invalid regex are
		// acceptable since this is a compile-time operation, not runtime.
		// We use Compile (not MustCompile) to avoid expected panics.
		_, _ = regexp.Compile("((?m)^" + pattern + "$)")
	})
}
