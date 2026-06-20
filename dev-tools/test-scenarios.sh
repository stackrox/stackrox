#!/usr/bin/env bash
set -euo pipefail
cd /Users/house/dev/stack/stackrox/.claude/worktrees/kind-local-dev
GOARCH="$(go env GOARCH)"
OUTDIR="bin/linux_${GOARCH}"
NS=stackrox
CTX=kind-stackrox-dev
REG=stackrox-dev-registry:5000
LDFLAGS="-s -w"

_ms() { python3 -c 'import time; print(int(time.time()*1000))'; }
_bk() {
    podman exec buildkitd buildctl build \
        --addr unix:///run/buildkit/buildkitd.sock \
        --frontend dockerfile.v0 \
        --local context=/context --local dockerfile=/context \
        --opt build-arg:TARGET_ARCH=${GOARCH} \
        --output "type=image,name=${REG}/main:$1,push=true,registry.insecure=true" \
        2>&1 | grep 'DONE' | tail -1
}
_go1() {
    GOOS=linux GOARCH=$GOARCH CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="$LDFLAGS" -o "${OUTDIR}/$1" "$2" 2>&1
}
_goall() {
    for e in central:./central migrator:./migrator compliance:./compliance/cmd/compliance kubernetes-sensor:./sensor/kubernetes sensor-upgrader:./sensor/upgrader admission-control:./sensor/admission-control config-controller:./config-controller; do
        _go1 "${e%%:*}" "${e##*:}"
    done
}

echo "=== SCENARIO TIMING RESULTS ==="
echo ""

# Warm up: no-change build to prime cache
_go1 central ./central >/dev/null
_goall >/dev/null

T=$(_ms); _go1 central ./central >/dev/null
S1=$(($(_ms)-T))
echo "S1  No-change compile (1 binary):          ${S1}ms"

echo '// s2' >> central/main.go
T=$(_ms); _go1 central ./central >/dev/null
S2=$(($(_ms)-T))
echo "S2  One-line change compile (1 binary):    ${S2}ms"
git checkout central/main.go 2>/dev/null

T=$(_ms); _goall >/dev/null
S3=$(($(_ms)-T))
echo "S3  No-change compile (all 7 binaries):    ${S3}ms"

echo '// s4' >> central/main.go
T=$(_ms); _goall >/dev/null
S4=$(($(_ms)-T))
echo "S4  One-change compile (all 7):            ${S4}ms"
git checkout central/main.go 2>/dev/null

T=$(_ms); _bk s5
S5=$(($(_ms)-T))
echo "S5  No-change image build+push:            ${S5}ms"

touch image/rhel/bin/central
T=$(_ms); _bk s6
S6=$(($(_ms)-T))
echo "S6  One-binary image build+push:           ${S6}ms"

T=$(_ms)
kubectl --context $CTX -n $NS set image deploy/central central=localhost:5000/main:s6 2>/dev/null
kubectl --context $CTX -n $NS delete pod -l app=central --grace-period=0 2>/dev/null
S7=$(($(_ms)-T))
echo "S7  kubectl set image + kill pod:          ${S7}ms"

T=$(_ms)
for i in $(seq 1 60); do
    ready=$(kubectl --context $CTX -n $NS get deploy/central -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
    [[ "$ready" == "1" ]] && break; sleep 1
done
S8=$(($(_ms)-T))
echo "S8  Pod restart until Ready:               ${S8}ms"

echo ""

# S9: FULL E2E — code change → compile → image → deploy → log visible
marker="SCENARIO-$(date +%s)-$$"
cat > central/dev_scenario_marker.go <<GOFILE
package main
import ("fmt"; "os")
func init() { fmt.Fprintln(os.Stderr, "$marker") }
GOFILE
trap 'rm -f central/dev_scenario_marker.go' EXIT

TAG="e2e-$(date +%s)"
T=$(_ms)
_go1 central ./central >/dev/null
T1=$(_ms)
cp ${OUTDIR}/central image/rhel/bin/central
_bk "$TAG" >/dev/null
T2=$(_ms)
kubectl --context $CTX -n $NS set image deploy/central "central=localhost:5000/main:${TAG}" 2>/dev/null
kubectl --context $CTX -n $NS delete pod -l app=central --grace-period=0 2>/dev/null
T3=$(_ms)
for i in $(seq 1 60); do
    kubectl --context $CTX -n $NS logs -l app=central --tail=500 2>/dev/null | grep -q "$marker" && break
    sleep 1
done
T4=$(_ms)
rm -f central/dev_scenario_marker.go

echo "S9  FULL E2E: code change → log visible"
echo "    compile:      $((T1 - T))ms"
echo "    image+push:   $((T2 - T1))ms"
echo "    kubectl:      $((T3 - T2))ms"
echo "    pod restart:  $((T4 - T3))ms"
echo "    TOTAL:        $((T4 - T))ms"
S9=$((T4 - T))

echo ""
echo "=== RESULTS ==="
echo ""
printf "| %-45s | %7s | %7s | %s |\n" "Scenario" "Time" "Target" "Status"
printf "| %-45s | %7s | %7s | %s |\n" "---" "---" "---" "---"
r() { [[ $1 -lt $2 ]] && echo "PASS" || echo "FAIL"; }
printf "| %-45s | %5sms | %5sms | %s |\n" "S1 No-change compile (1 binary)" "$S1" "5000" "$(r $S1 5000)"
printf "| %-45s | %5sms | %5sms | %s |\n" "S2 One-line change compile (1 binary)" "$S2" "10000" "$(r $S2 10000)"
printf "| %-45s | %5sms | %5sms | %s |\n" "S3 No-change compile (all 7)" "$S3" "15000" "$(r $S3 15000)"
printf "| %-45s | %5sms | %5sms | %s |\n" "S4 One-change compile (all 7)" "$S4" "20000" "$(r $S4 20000)"
printf "| %-45s | %5sms | %5sms | %s |\n" "S5 No-change image build+push" "$S5" "2000" "$(r $S5 2000)"
printf "| %-45s | %5sms | %5sms | %s |\n" "S6 One-binary image build+push" "$S6" "8000" "$(r $S6 8000)"
printf "| %-45s | %5sms | %5sms | %s |\n" "S7 kubectl set image + kill pod" "$S7" "3000" "$(r $S7 3000)"
printf "| %-45s | %5sms | %5sms | %s |\n" "S8 Pod restart until Ready" "$S8" "20000" "$(r $S8 20000)"
printf "| %-45s | %5sms | %5sms | %s |\n" "S9 Full E2E: change → log visible" "$S9" "30000" "$(r $S9 30000)"

