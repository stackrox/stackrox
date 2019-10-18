MINUTE=60
FIVE_MINUTE=300
BENCH_REGEX=$1
BENCH_PATH=$2
echo "starting benchmark"
START_TIME=$(date +%s)
go test -bench "$BENCH_REGEX" "$BENCH_PATH"
echo "finished benchmark, sleeping to allow time for metrics to be updated"
sleep 70
DURATION="$(($(date +%s) - $START_TIME))"
DURATION=$(($DURATION>$FIVE_MINUTE ? $DURATION : $FIVE_MINUTE))
DURATION=$(($DURATION / $MINUTE))
export DURATION_MIN=$DURATION
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
echo "storing monitoring graphs"
"$DIR"/../../monitoring/grafana/fetch-core-images.sh $DURATION
echo "finished storing monitoring graphs"