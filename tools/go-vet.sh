#!/usr/bin/env bash
# go vet variant that respects NOVET annotations

ignored=0
total=0
status=0

# Check if this is for a release tag. This matters because we don't vet test files on release tags.
is_release=0
# Not the most robust way of checking, but works given our make invocation.
# In any case, we default to is_release=0, which means we check test files too,
# so if someone changes the invocation, they will find out since vet won't pass
# on test files on release tags.
if [[ "$1" == "-tags" && "$2" == "release" ]]; then
  is_release=1
fi

while read -r line; do
    if [[ "$line" =~ ^exit\ status\ ([[:digit:]]+)$ ]]; then
        status="${BASH_REMATCH[1]}"
        continue
    fi

    line_to_echo="${line}"
    if [[ "$line" != \#* ]]; then # Ignore comment lines
        total=$((total + 1))
        if [[ "$line" =~ ^([^:]+):([[:digit:]]+): ]]; then
            filename="${BASH_REMATCH[1]}"
            lineno="${BASH_REMATCH[2]}"
            line_in_file=$(tail -n+"$lineno" "$filename" | head -n1)
            if [[ "$line_in_file" =~ //\ NOVET$ ]]; then
                line_to_echo="(${line} -- IGNORED due to NOVET annotation)"
                ignored=$((ignored + 1))
            fi
            if (( is_release )); then
                if [[ "$filename" =~ _test\.go$ ]]; then
                    line_to_echo="(${line} -- IGNORED because it's a test file and we're running on release tags)"
                    ignored=$((ignored + 1))
                fi
            fi
        fi
    fi
    echo >&2 "${line_to_echo}"
done < <(go vet -all -printfuncs Print,Printf,Println,Debug,Debugf,Info,Infof,Warn,Warnf,Error,Errorf "$@" 2>&1; echo "exit status $?")

echo "Found ${total} errors, ignored ${ignored}"
if (( total == ignored )); then
    exit 0
else
    exit "$status"
fi
