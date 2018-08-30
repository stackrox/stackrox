#!/usr/bin/env bash
# go vet variant that respects NOVET annotations

ignored=0
total=0
status=0

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
