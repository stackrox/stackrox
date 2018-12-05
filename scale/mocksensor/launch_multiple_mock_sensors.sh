#!/usr/bin/env bash
set -eu

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

let instances=1
params=""

for i in "$@"
do
case $i in
 -instances=*)
 instances=${i#*=}
 ;;
 *)
 params="${params} ${i}"
esac
done

for (( i=1;i<=${instances};i++ ))
do
   clusterName=mock-sensor-${i}
   deploymentName=mock-sensor-${i}
   secretName=sensor-tls-${i}

   cmd="$DIR/launch_mock_sensor.sh ${i}"

   if [[ ! -z "${params}" ]]
   then
        cmd+=${params}
   fi

   $cmd
done
