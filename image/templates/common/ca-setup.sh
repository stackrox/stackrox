#!/usr/bin/env bash

{{  $kubeCmd := "" -}}
{{- $secretName := "additional-ca" -}}
{{- if .K8sConfig -}}
{{- $kubeCmd = .K8sConfig.Command -}}
{{- else -}}
{{- $kubeCmd = .K8sCommand -}}
{{- $secretName = "additional-ca-sensor" -}}
{{- end -}}
{{- if not $kubeCmd -}}
{{- $kubeCmd = "kubectl" -}}
{{- end -}}

KUBE_COMMAND=${KUBE_COMMAND:-{{$kubeCmd}}}

update=0

function create_or_replace() {
	cmd=create
	(( ! update )) || cmd=replace
	${KUBE_COMMAND} create "$@" --dry-run -o yaml | ${KUBE_COMMAND} "$cmd" -f -
}

function label() {
	extra_args=()
	(( ! update )) || extra_args+=(--overwrite)
	${KUBE_COMMAND} label "${extra_args[@]}" "$@"
}

function usage {
	echo "usage:"
	echo "    $(basename "$0") [-u] -f file"
	echo "    $(basename "$0") [-u] -d dir"
	echo
	echo "The argument may be:"
	echo "  - a single file"
	echo "  - a directory (all files ending in .crt and .pem will be added)"
	echo "Each file must contain exactly one PEM-encoded certificate."
	echo
	echo "If the -u (update) argument is passed, the existing additional CAs will be"
	echo "replaced."
	exit 1
}

function create_ns {
	${KUBE_COMMAND} get ns "stackrox" > /dev/null 2>&1 || ${KUBE_COMMAND} create ns "stackrox"
}

function create_file {
	local file="$1"
	create_or_replace secret -n "stackrox" generic {{$secretName}} --from-file="ca.crt=$file"
	label -n "stackrox" secret/{{$secretName}} app.kubernetes.io/name=stackrox
}

function create_directory {
	local dir="$1"
	echo "The following certificates will be used as additional CAs:"
	from_file_args=()
	for f in "$dir"/*.crt "$dir"/*.pem; do
    	if [ -f "$f" ] ; then
    		from_file_args+=("--from-file=$(basename "$f")=$f")
			echo "  - $f"
		fi
	done
	if [ "${#from_file_args[@]}" -eq 0 ]; then
		echo "Error: No filenames ending in \".crt\" or \".pem\" in $dir. Please add some."
		exit 2
	fi
	create_or_replace secret -n "stackrox" generic {{$secretName}} "${from_file_args[@]}"
	label -n "stackrox" secret/{{$secretName}} app.kubernetes.io/name=stackrox
}

[[ "$#" -ge 2 ]] || usage

file_name=
dir_name=

while [[ "$#" -gt 0 ]]; do
	arg="$1"
	shift

	case "$arg" in
	-f)
		[[ -z "$file_name" ]] || usage
		file_name="$1"
		shift
		;;
	-d)
		[[ -z "$dir_name" ]] || usage
		dir_name="$1"
		shift
		;;
	-u)
		(( ! update )) || usage
		update=1
		;;
	*)
		usage
	esac
done

create_ns

if [[ -n "$file_name" && -z "$dir_name" ]]; then
	create_file "$file_name"
elif [[ -n "$dir_name" && -z "$file_name" ]]; then
	create_directory "$dir_name"
else
	usage
fi
