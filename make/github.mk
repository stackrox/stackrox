# github.mk
# Helpers for fetching released binaries from github projects.

# For usage instructions, see uses of this macro elsewhere, since there is no
# single standard for architecture naming.
GET_GITHUB_RELEASE_FN = get_github_release() { \
	[[ -x $${1} ]] || { \
		set -euo pipefail ;\
		echo "+ $${1}" ;\
		mkdir -p bin ;\
		attempts=5 ;\
		for i in $$(seq $$attempts); do \
			curl --silent --show-error --fail --location --output "$${1}" "$${2}" && break ;\
			[[ $$i -eq $$attempts ]] && exit 1; \
			echo "Retrying after $$((i*i)) seconds..."; \
			sleep "$$((i*i))"; \
		done ;\
		[[ $$(uname -s) != Darwin ]] || xattr -c "$${1}" ;\
		chmod +x "$${1}" ;\
	} \
}
