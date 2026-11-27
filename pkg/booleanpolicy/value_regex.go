package booleanpolicy

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/stackrox/rox/pkg/search/predicate/basematchers"
	"github.com/stackrox/rox/pkg/signatures"
)

const (
	ipv4Regex = "(\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})"
	ipv6Regex = "((?:[0-9A-Fa-f]{1,4}))((?::[0-9A-Fa-f]{1,4}))*::((?:[0-9A-Fa-f]{1,4}))((?::[0-9A-Fa-f]{1,4}))*|((?:[0-9A-Fa-f]{1,4}))((?::[0-9A-Fa-f]{1,4})){7}"
)

var (
	keyValueValueRegex          = createRegex("[^=]+=.*")
	booleanValueRegex           = createRegex("(?i:(true|false))")
	stringValueRegex            = createRegex(".*[^[:space:]]+.*")
	integerValueRegex           = createRegex("([[:digit:]]+)")
	comparatorDecimalValueRegex = createRegex(`(<|>|<=|>=)?[[:space:]]*[[:digit:]]*\.?[[:digit:]]+`)
	// Whole-number percent in [0, 100], with an optional numeric comparator.
	// ">" and ">=" stop at 99. Combined with 100 they scale to a probability
	// comparison above 1, and stored EPSS probability is at most 1.
	comparatorEPSSPercentRegex               = createRegex(`(?:(?:<=|<)[[:space:]]*(?:100|[1-9][0-9]|[0-9])|(?:>=|>)[[:space:]]*(?:[1-9][0-9]|[0-9])|(?:100|[1-9][0-9]|[0-9]))`)
	environmentVariableWithSourceStrictRegex = createRegex("(?i:(UNSET|RAW|UNKNOWN|^)=([^=]*)=.*)|((SECRET_KEY|CONFIG_MAP_KEY|FIELD|RESOURCE_FIELD)=([^=]*)=$)")
	environmentVariableWithSourceRegex       = createRegex("(?i:(UNSET|RAW|SECRET_KEY|CONFIG_MAP_KEY|FIELD|RESOURCE_FIELD|UNKNOWN|^)=[^=]*=.*)")
	dockerfileLineValueRegex                 = createRegex("(?i:(ADD|ARG|CMD|COPY|ENTRYPOINT|ENV|EXPOSE|FROM|LABEL|MAINTAINER|ONBUILD|RUN|STOPSIGNAL|USER|VOLUME|WORKDIR|^)=).*")
	dockerfileLineValueRegexNoFrom           = createRegex("(?i:(ADD|ARG|CMD|COPY|ENTRYPOINT|ENV|EXPOSE|LABEL|MAINTAINER|ONBUILD|RUN|STOPSIGNAL|USER|VOLUME|WORKDIR|^)=).*")
	dropCapabilitiesValueRegex               = createRegex("(?i:(ALL|AUDIT_CONTROL|AUDIT_READ|AUDIT_WRITE|BLOCK_SUSPEND|CHOWN|DAC_OVERRIDE|DAC_READ_SEARCH|FOWNER|FSETID|IPC_LOCK|IPC_OWNER|KILL|LEASE|LINUX_IMMUTABLE|MAC_ADMIN|MAC_OVERRIDE|MKNOD|NET_ADMIN|NET_BIND_SERVICE|NET_BROADCAST|NET_RAW|SETGID|SETFCAP|SETPCAP|SETUID|SYS_ADMIN|SYS_BOOT|SYS_CHROOT|SYS_MODULE|SYS_NICE|SYS_PACCT|SYS_PTRACE|SYS_RAWIO|SYS_RESOURCE|SYS_TIME|SYS_TTY_CONFIG|SYSLOG|WAKE_ALARM))")
	addCapabilitiesValueRegex                = createRegex("(?i:(AUDIT_CONTROL|AUDIT_READ|AUDIT_WRITE|BLOCK_SUSPEND|CHOWN|DAC_OVERRIDE|DAC_READ_SEARCH|FOWNER|FSETID|IPC_LOCK|IPC_OWNER|KILL|LEASE|LINUX_IMMUTABLE|MAC_ADMIN|MAC_OVERRIDE|MKNOD|NET_ADMIN|NET_BIND_SERVICE|NET_BROADCAST|NET_RAW|SETGID|SETFCAP|SETPCAP|SETUID|SYS_ADMIN|SYS_BOOT|SYS_CHROOT|SYS_MODULE|SYS_NICE|SYS_PACCT|SYS_PTRACE|SYS_RAWIO|SYS_RESOURCE|SYS_TIME|SYS_TTY_CONFIG|SYSLOG|WAKE_ALARM))")
	rbacPermissionValueRegex                 = createRegex("(?i:DEFAULT|ELEVATED_IN_NAMESPACE|ELEVATED_CLUSTER_WIDE|CLUSTER_ADMIN)")
	portExposureValueRegex                   = createRegex("(?i:UNSET|EXTERNAL|NODE|HOST|INTERNAL|ROUTE)")
	kubernetesResourceValueRegex             = createRegex(`(?i:PODS_EXEC|PODS_PORTFORWARD|PODS_ATTACH)`)
	mountPropagationValueRegex               = createRegex("(?i:NONE|HOSTTOCONTAINER|BIDIRECTIONAL)")
	seccompProfileTypeValueRegex             = createRegex(`(?i:UNCONFINED|RUNTIME_DEFAULT|LOCALHOST)`)
	severityValueRegex                       = createRegex(`(<|>|<=|>=)?[[:space:]]*(?i:UNKNOWN|LOW|MODERATE|IMPORTANT|CRITICAL)`)
	auditEventAPIVerbValueRegex              = createRegex(`(?i:CREATE|DELETE|GET|PATCH|UPDATE)`)
	auditEventResourceValueRegex             = createRegex(`(?i:SECRETS|CONFIGMAPS|CLUSTER_ROLES|CLUSTER_ROLE_BINDINGS|NETWORK_POLICIES|SECURITY_CONTEXT_CONSTRAINTS|EGRESS_FIREWALLS|EVENTS)`)
	kubernetesNameRegex                      = createRegex(`(?i:[a-z0-9])(?i:[-:a-z0-9]*[a-z0-9])?`)
	ipAddressValueRegex                      = createRegex(fmt.Sprintf(`(%s)|(%s)`, ipv4Regex, ipv6Regex))
	signatureIntegrationIDValueRegex         = createRegex(regexp.QuoteMeta(signatures.SignatureIntegrationIDPrefix) + "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}")
	fileOperationRegex                       = createRegex(`(?i:OPEN|CREATE|RENAME|UNLINK|OWNERSHIP_CHANGE|PERMISSION_CHANGE|XATTR_CHANGE)`)
)

func createRegex(s string) *regexp.Regexp {
	// set multiline anchor for beginning and end of regex
	return regexp.MustCompile("((?m)^" + s + "$)")
}

// validateEPSSPercent checks a policy EPSS value. The value must match the
// whole-number percent pattern. A value that does not match is parsed so
// ">" and ">=" combined with 100 can return a specific error: those
// comparisons scale past a probability of 1, and stored EPSS is at most 1.
func validateEPSSPercent(_ *validateConfiguration, value string) (bool, error) {
	if comparatorEPSSPercentRegex.MatchString(value) {
		return true, nil
	}
	prefix, number := basematchers.ParseNumericPrefix(strings.TrimSpace(value))
	if number == "100" && (prefix == basematchers.GreaterThan || prefix == basematchers.GreaterThanOrEqualTo) {
		return false, errors.New(`">" and ">=" cannot be combined with 100 because EPSS probability is at most 1`)
	}
	return false, fmt.Errorf("must match %q", comparatorEPSSPercentRegex.String())
}
