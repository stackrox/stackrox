#!/usr/bin/env bash
set -eu

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

   cmd="./launch_mock_sensor.sh -cluster-name=${clusterName} -deployment-name=${deploymentName} -secret-name=${secretName}"

   if [[ ! -z "${params}" ]]
   then
        cmd+=${params}
   fi

   $cmd
   sleep 60
done