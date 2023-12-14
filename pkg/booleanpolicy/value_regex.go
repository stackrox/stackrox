package booleanpolicy

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/pkg/signatures"
)

const (
	ipv4Regex = "(\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})"
	ipv6Regex = "((?:[0-9A-Fa-f]{1,4}))((?::[0-9A-Fa-f]{1,4}))*::((?:[0-9A-Fa-f]{1,4}))((?::[0-9A-Fa-f]{1,4}))*|((?:[0-9A-Fa-f]{1,4}))((?::[0-9A-Fa-f]{1,4})){7}"
)

var (
	keyValueValueRegex                       = createRegex("[^=]+=.*")
	booleanValueRegex                        = createRegex("(?i:(true|false))")
	stringValueRegex                         = createRegex(".*[^[:space:]]+.*")
	integerValueRegex                        = createRegex("([[:digit:]]+)")
	comparatorDecimalValueRegex              = createRegex(`(<|>|<=|>=)?[[:space:]]*[[:digit:]]*\.?[[:digit:]]+`)
	environmentVariableWithSourceStrictRegex = createRegex("(?i:(UNSET|RAW|UNKNOWN|^)=([^=]*)=.*)|((SECRET_KEY|CONFIG_MAP_KEY|FIELD|RESOURCE_FIELD)=([^=]*)=$)")
	environmentVariableWithSourceRegex       = createRegex("(?i:(UNSET|RAW|SECRET_KEY|CONFIG_MAP_KEY|FIELD|RESOURCE_FIELD|UNKNOWN|^)=[^=]*=.*)")
	dockerfileLineValueRegex                 = createRegex("(?i:(ADD|ARG|CMD|COPY|ENTRYPOINT|ENV|EXPOSE|FROM|LABEL|MAINTAINER|ONBUILD|RUN|STOPSIGNAL|USER|VOLUME|WORKDIR|^)=).*")
	dockerfileLineValueRegexNoFrom           = createRegex("(?i:(ADD|ARG|CMD|COPY|ENTRYPOINT|ENV|EXPOSE|LABEL|MAINTAINER|ONBUILD|RUN|STOPSIGNAL|USER|VOLUME|WORKDIR|^)=).*")
	dropCapabilitiesValueRegex               = createRegex("(?i:(ALL|AUDIT_CONTROL|AUDIT_READ|AUDIT_WRITE|BLOCK_SUSPEND|CHOWN|DAC_OVERRIDE|DAC_READ_SEARCH|FOWNER|FSETID|IPC_LOCK|IPC_OWNER|KILL|LEASE|LINUX_IMMUTABLE|MAC_ADMIN|MAC_OVERRIDE|MKNOD|NET_ADMIN|NET_BIND_SERVICE|NET_BROADCAST|NET_RAW|SETGID|SETFCAP|SETPCAP|SETUID|SYS_ADMIN|SYS_BOOT|SYS_CHROOT|SYS_MODULE|SYS_NICE|SYS_PACCT|SYS_PTRACE|SYS_RAWIO|SYS_RESOURCE|SYS_TIME|SYS_TTY_CONFIG|SYSLOG|WAKE_ALARM))")
	addCapabilitiesValueRegex                = createRegex("(?i:(AUDIT_CONTROL|AUDIT_READ|AUDIT_WRITE|BLOCK_SUSPEND|CHOWN|DAC_OVERRIDE|DAC_READ_SEARCH|FOWNER|FSETID|IPC_LOCK|IPC_OWNER|KILL|LEASE|LINUX_IMMUTABLE|MAC_ADMIN|MAC_OVERRIDE|MKNOD|NET_ADMIN|NET_BIND_SERVICE|NET_BROADCAST|NET_RAW|SETGID|SETFCAP|SETPCAP|SETUID|SYS_ADMIN|SYS_BOOT|SYS_CHROOT|SYS_MODULE|SYS_NICE|SYS_PACCT|SYS_PTRACE|SYS_RAWIO|SYS_RESOURCE|SYS_TIME|SYS_TTY_CONFIG|SYSLOG|WAKE_ALARM))")
	rbacPermissionValueRegex                 = createRegex("(?i:DEFAULT|ELEVATED_IN_NAMESPACE|ELEVATED_CLUSTER_WIDE|CLUSTER_ADMIN)")
	portExposureValueRegex                   = createRegex("(?i:UNSET|EXTERNAL|NODE|HOST|INTERNAL|ROUTE)")
	kubernetesAPIVerbValueRegex              = createRegex(`(?i:CREATE)`)
	kubernetesResourceValueRegex             = createRegex(`(?i:PODS_EXEC|PODS_PORTFORWARD)`)
	mountPropagationValueRegex               = createRegex("(?i:NONE|HOSTTOCONTAINER|BIDIRECTIONAL)")
	seccompProfileTypeValueRegex             = createRegex(`(?i:UNCONFINED|RUNTIME_DEFAULT|LOCALHOST)`)
	severityValueRegex                       = createRegex(`(<|>|<=|>=)?[[:space:]]*(?i:UNKNOWN|LOW|MODERATE|IMPORTANT|CRITICAL)`)
	auditEventAPIVerbValueRegex              = createRegex(`(?i:CREATE|DELETE|GET|PATCH|UPDATE)`)
	auditEventResourceValueRegex             = createRegex(`(?i:SECRETS|CONFIGMAPS|CLUSTER_ROLES|CLUSTER_ROLE_BINDINGS|NETWORK_POLICIES|SECURITY_CONTEXT_CONSTRAINTS|EGRESS_FIREWALLS)`)
	kubernetesNameRegex                      = createRegex(`(?i:[a-z0-9])(?i:[-:a-z0-9]*[a-z0-9])?`)
	ipAddressValueRegex                      = createRegex(fmt.Sprintf(`(%s)|(%s)`, ipv4Regex, ipv6Regex))
	signatureIntegrationIDValueRegex         = createRegex(regexp.QuoteMeta(signatures.SignatureIntegrationIDPrefix) + "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}")
)

func createRegex(s string) *regexp.Regexp {
	// set multiline anchor for beginning and end of regex
	return regexp.MustCompile("((?m)^" + s + "$)")
}
