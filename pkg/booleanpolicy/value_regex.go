package booleanpolicy

import (
	"fmt"
	"regexp"
	"sync"
)

const (
	ipv4Regex = "(\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})"
	ipv6Regex = "((?:[0-9A-Fa-f]{1,4}))((?::[0-9A-Fa-f]{1,4}))*::((?:[0-9A-Fa-f]{1,4}))((?::[0-9A-Fa-f]{1,4}))*|((?:[0-9A-Fa-f]{1,4}))((?::[0-9A-Fa-f]{1,4})){7}"
)

// signatureIntegrationIDPrefix is inlined from pkg/signatures.SignatureIntegrationIDPrefix
// to avoid importing pkg/signatures which transitively pulls in cosign/sigstore/rekor (~2 MB init).
const signatureIntegrationIDPrefix = "io.stackrox.signatureintegration."

var (
	keyValueValueRegex                       = lazyRegex("[^=]+=.*")
	booleanValueRegex                        = lazyRegex("(?i:(true|false))")
	stringValueRegex                         = lazyRegex(".*[^[:space:]]+.*")
	integerValueRegex                        = lazyRegex("([[:digit:]]+)")
	comparatorDecimalValueRegex              = lazyRegex(`(<|>|<=|>=)?[[:space:]]*[[:digit:]]*\.?[[:digit:]]+`)
	environmentVariableWithSourceStrictRegex = lazyRegex("(?i:(UNSET|RAW|UNKNOWN|^)=([^=]*)=.*)|((SECRET_KEY|CONFIG_MAP_KEY|FIELD|RESOURCE_FIELD)=([^=]*)=$)")
	environmentVariableWithSourceRegex       = lazyRegex("(?i:(UNSET|RAW|SECRET_KEY|CONFIG_MAP_KEY|FIELD|RESOURCE_FIELD|UNKNOWN|^)=[^=]*=.*)")
	dockerfileLineValueRegex                 = lazyRegex("(?i:(ADD|ARG|CMD|COPY|ENTRYPOINT|ENV|EXPOSE|FROM|LABEL|MAINTAINER|ONBUILD|RUN|STOPSIGNAL|USER|VOLUME|WORKDIR|^)=).*")
	dockerfileLineValueRegexNoFrom           = lazyRegex("(?i:(ADD|ARG|CMD|COPY|ENTRYPOINT|ENV|EXPOSE|LABEL|MAINTAINER|ONBUILD|RUN|STOPSIGNAL|USER|VOLUME|WORKDIR|^)=).*")
	dropCapabilitiesValueRegex               = lazyRegex("(?i:(ALL|AUDIT_CONTROL|AUDIT_READ|AUDIT_WRITE|BLOCK_SUSPEND|CHOWN|DAC_OVERRIDE|DAC_READ_SEARCH|FOWNER|FSETID|IPC_LOCK|IPC_OWNER|KILL|LEASE|LINUX_IMMUTABLE|MAC_ADMIN|MAC_OVERRIDE|MKNOD|NET_ADMIN|NET_BIND_SERVICE|NET_BROADCAST|NET_RAW|SETGID|SETFCAP|SETPCAP|SETUID|SYS_ADMIN|SYS_BOOT|SYS_CHROOT|SYS_MODULE|SYS_NICE|SYS_PACCT|SYS_PTRACE|SYS_RAWIO|SYS_RESOURCE|SYS_TIME|SYS_TTY_CONFIG|SYSLOG|WAKE_ALARM))")
	addCapabilitiesValueRegex                = lazyRegex("(?i:(AUDIT_CONTROL|AUDIT_READ|AUDIT_WRITE|BLOCK_SUSPEND|CHOWN|DAC_OVERRIDE|DAC_READ_SEARCH|FOWNER|FSETID|IPC_LOCK|IPC_OWNER|KILL|LEASE|LINUX_IMMUTABLE|MAC_ADMIN|MAC_OVERRIDE|MKNOD|NET_ADMIN|NET_BIND_SERVICE|NET_BROADCAST|NET_RAW|SETGID|SETFCAP|SETPCAP|SETUID|SYS_ADMIN|SYS_BOOT|SYS_CHROOT|SYS_MODULE|SYS_NICE|SYS_PACCT|SYS_PTRACE|SYS_RAWIO|SYS_RESOURCE|SYS_TIME|SYS_TTY_CONFIG|SYSLOG|WAKE_ALARM))")
	rbacPermissionValueRegex                 = lazyRegex("(?i:DEFAULT|ELEVATED_IN_NAMESPACE|ELEVATED_CLUSTER_WIDE|CLUSTER_ADMIN)")
	portExposureValueRegex                   = lazyRegex("(?i:UNSET|EXTERNAL|NODE|HOST|INTERNAL|ROUTE)")
	kubernetesResourceValueRegex             = lazyRegex(`(?i:PODS_EXEC|PODS_PORTFORWARD|PODS_ATTACH)`)
	mountPropagationValueRegex               = lazyRegex("(?i:NONE|HOSTTOCONTAINER|BIDIRECTIONAL)")
	seccompProfileTypeValueRegex             = lazyRegex(`(?i:UNCONFINED|RUNTIME_DEFAULT|LOCALHOST)`)
	severityValueRegex                       = lazyRegex(`(<|>|<=|>=)?[[:space:]]*(?i:UNKNOWN|LOW|MODERATE|IMPORTANT|CRITICAL)`)
	auditEventAPIVerbValueRegex              = lazyRegex(`(?i:CREATE|DELETE|GET|PATCH|UPDATE)`)
	auditEventResourceValueRegex             = lazyRegex(`(?i:SECRETS|CONFIGMAPS|CLUSTER_ROLES|CLUSTER_ROLE_BINDINGS|NETWORK_POLICIES|SECURITY_CONTEXT_CONSTRAINTS|EGRESS_FIREWALLS)`)
	kubernetesNameRegex                      = lazyRegex(`(?i:[a-z0-9])(?i:[-:a-z0-9]*[a-z0-9])?`)
	ipAddressValueRegex                      = lazyRegex(fmt.Sprintf(`(%s)|(%s)`, ipv4Regex, ipv6Regex))
	signatureIntegrationIDValueRegex         = lazyRegex(regexp.QuoteMeta(signatureIntegrationIDPrefix) + "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}")
	fileOperationRegex                       = lazyRegex(`(?i:OPEN|CREATE|RENAME|UNLINK|OWNERSHIP_CHANGE|PERMISSION_CHANGE)`)
)

func lazyRegex(s string) func() *regexp.Regexp {
	return sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile("((?m)^" + s + "$)")
	})
}
