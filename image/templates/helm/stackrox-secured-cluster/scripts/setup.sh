#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

function parseYaml {
   local prefix=$2
   local s='[[:space:]]*' w='[a-zA-Z0-9_]*' fs=$(echo @|tr @ '\034')
   sed -ne "s|^\($s\):|\1|" \
        -e "s|^\($s\)\($w\)$s:$s[\"']\(.*\)[\"']$s\$|\1$fs\2$fs\3|p" \
        -e "s|^\($s\)\($w\)$s:$s\(.*\)$s\$|\1$fs\2$fs\3|p"  "${1}" |
   awk -F$fs '{
      indent = length($1)/2;
      vname[indent] = $2;
      for (i in vname) {if (i > indent) {delete vname[i]}}
      if (length($3) > 0) {
         vn=""; for (i=0; i<indent; i++) {vn=(vn)(vname[i])("_")}
         printf("%s%s%s=\"%s\"\n", "'${prefix}'",vn, $2, $3);
      }
   }'
}

function printHelp {
      echo "Usage:"
      echo "./setup.sh [options]"
      echo "Options:"
      echo "-h             help for setup.sh."
      echo "-f string      add Helm Cluster with values specified in the file."
      echo "-e string      endpoint for service to contact (default "localhost:8443")"
}


valuesFile="${DIR}/../values.yaml"
endpoint="localhost:8443"

while getopts "hf:e:?:" opt; do
  case ${opt} in
    h )
      printHelp
      exit 0
      ;;
    f )
      valuesFile=$OPTARG
      ;;
    e )
      endpoint=$OPTARG
      ;;
    \? )
      echo "Invalid option: $OPTARG" 1>&2
      exit 1
      ;;
    : )
      echo "Invalid option: $OPTARG requires an argument" 1>&2
      exit 1
      ;;
  esac
done
shift $((OPTIND -1))


# parse values.yaml to generate config variables
parseYaml "${valuesFile}" > "${DIR}"/config.sh

. "${DIR}"/config.sh

echo "Adding cluster ${cluster_name} ..."
"${DIR}"/add-cluster.sh "${endpoint}"
