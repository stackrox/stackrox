#!/usr/bin/env bash
set -eu

# This launches mock_sensor with the tag defined by `make tag`.
# Any arguments passed to this script are passed on to the mocksensor program.
# Example: ./launch_mock_sensor.sh -max-deployments 100 will launch mocksensor with the args -max-deployments 100.

tag="$(make -C ../../ tag)"
echo "Launching mock sensor with tag: ${tag}"
if [[ "$#" -gt 0 ]]; then
  for (( i=$#;i>0;i-- ));do
  sed -i .bak 's@- /mocksensor@- /mocksensor\
          - "'"${!i}"'"@' mocksensor.yaml
  done
fi
sed -i .bak 's@image: .*@image: stackrox/scale:'"${tag}"'@' mocksensor.yaml
kubectl -n stackrox delete deploy/sensor || true
sleep 5
kubectl create -f mocksensor.yaml
git checkout -- mocksensor.yaml
