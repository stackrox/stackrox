#!/usr/bin/env bash
set -eu
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

# This launches mock_sensor with the tag defined by `make tag`.
# Any arguments passed to this script are passed on to the mocksensor program.
# Example: ./launch_mock_collector.sh -max-collectors 100 -max-processes 1000 will launch
# mockcollector with the args -max-collectors 100 and -max-processes 1000.

tag="$(git describe --tags --abbrev=10 --dirty)"
echo "Launching mock collector with tag: ${tag}"
if [[ "$#" -gt 0 ]]; then
  for (( i=$#;i>0;i-- ));do
  sed -i.bak 's@- /mockcollector@- /mockcollector \
          - "'"${!i}"'"@' ${DIR}/mockcollector.yaml 2>/dev/null || true
  done
fi
sed -i.bak 's@image: .*@image: stackrox/scale:'"${tag}"'@' ${DIR}/mockcollector.yaml
kubectl -n stackrox delete daemonset/collector || true
sleep 5
kubectl create -f ${DIR}/mockcollector.yaml
git checkout -- ${DIR}/mockcollector.yaml
