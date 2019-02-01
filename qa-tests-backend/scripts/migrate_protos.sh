#! /bin/bash

JAVA_PATH=src/main/proto/

for file in $(find ../proto/*); do
    if [[ -d $file ]]; then
        dir=${file#"../proto/"}
        mkdir -p "${JAVA_PATH}${dir}"
        echo "${JAVA_PATH}${dir}"
    fi
done

for file in $(find ../proto/*); do
    if [[ -f $file ]]; then
        java_file=${file#"../proto/"}
        sed -e 's/\[[^][]*\]//g' "$file" | sed -e 's/\[[^][]*\]//g' | sed '/gogo/d' > "${JAVA_PATH}${java_file}"
    fi
done
