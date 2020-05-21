package booleanpolicy

import (
	"regexp"
)

var (
	keyValueValueRegex                 = createRegex("[^=]+=.*")
	booleanValueRegex                  = createRegex("(?i:(true|false))")
	stringValueRegex                   = createRegex(".*[^[:space:]]+.*")
	integerValueRegex                  = createRegex("([[:digit:]]+)")
	comparatorDecimalValueRegex        = createRegex(`(<|>|<=|>=)?[[:space:]]*[[:digit:]]*\.?[[:digit:]]+`)
	environmentVariableWithSourceRegex = createRegex("((UNSET|RAW|SECRET_KEY|CONFIG_MAP_KEY|FIELD|RESOURCE_FIELD|UNKNOWN)=.*=.*)")
	dockerfileLineValueRegex           = createRegex("(?i:(ADD|ARG|CMD|COPY|ENTRYPOINT|ENV|EXPOSE|FROM|LABEL|MAINTAINER|ONBUILD|RUN|STOPSIGNAL|USER|VOLUME|WORKDIR)=).*[^[:space:]]+.*")
	capabilitiesValueRegex             = createRegex("(?i:CAP_(AUDIT_CONTROL|AUDIT_READ|AUDIT_WRITE|BLOCK_SUSPEND|CHOWN|DAC_OVERRIDE|DAC_READ_SEARCH|FOWNER|FSETID|IPC_LOCK|IPC_OWNER|KILL|LEASE|LINUX_IMMUTABLE|MAC_ADMIN|MAC_OVERRIDE|MKNOD|NET_ADMIN|NET_BIND_SERVICE|NET_BROADCAST|NET_RAW|SETGID|SETFCAP|SETPCAP|SETUID|SYS_ADMIN|SYS_BOOT|SYS_CHROOT|SYS_MODULE|SYS_NICE|SYS_PACCT|SYS_PTRACE|SYS_RAWIO|SYS_RESOURCE|SYS_TIME|SYS_TTY_CONFIG|SYSLOG|WAKE_ALARM))")
	rbacPermissionValueRegex           = createRegex("(?i:DEFAULT|ELEVATED_IN_NAMESPACE|ELEVATED_CLUSTER_WIDE|CLUSTER_ADMIN)")
	portExposureValueRegex             = createRegex("(?i:UNSET|EXTERNAL|NODE|HOST|INTERNAL)")
)

func createRegex(s string) *regexp.Regexp {
	// set multiline anchor for beginning and end of regex
	return regexp.MustCompile("((?m)^" + s + "$)")
}
