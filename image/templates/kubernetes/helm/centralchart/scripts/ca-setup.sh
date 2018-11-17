#!/usr/bin/env bash

function usage {
	echo "usage:"
	echo "    ca-setup.sh -f file"
	echo "    ca-setup.sh -d dir"
	echo
	echo "The argument may be:"
	echo "  - a single file"
	echo "  - a directory (all files ending in .crt will be added)"
	echo "Each file must contain exactly one PEM-encoded certificate."
	exit 1
}

function create_ns {
    {{.K8sConfig.Command}} get ns "{{.K8sConfig.Namespace}}" > /dev/null || {{.K8sConfig.Command}} create ns "{{.K8sConfig.Namespace}}"
}
function create_file {
    local file="$1"
    {{.K8sConfig.Command}} create secret -n "{{.K8sConfig.Namespace}}" generic additional-ca --from-file="ca.crt=$file"
}

function create_directory {
    local dir="$1"
    echo "The following certificates will be used as additional CAs:"
    count=0
    for f in $dir/*.crt; do
        if [ -f "$f" ] ; then
            count=$((count+1))
            echo "  - $f"
        fi
    done
    if [ "$count" -eq 0 ]; then
        echo "Error: No filenames ending in \".crt\" in $dir. Please add some."
        exit 2
    fi
    {{.K8sConfig.Command}} create secret -n "{{.K8sConfig.Namespace}}" generic additional-ca --from-file="$dir/"
}

if [[ "$#" -lt 2 ]]; then
    usage
fi

create_ns

if [ "$1" = "-f" ]; then
    create_file "$2"
elif [ "$1" = "-d" ]; then
    create_directory "$2"
else
    usage
fi
