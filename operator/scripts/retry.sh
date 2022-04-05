#!/bin/sh

DELAY_BETWEEN_ATTEMPTS=2 # Seconds

eecho() {
  echo "$@" >&2
}

die() {
  eecho "$@"
  exit 1
}

if [ $# -lt 1 ]; then
    die "Usage: $0 <number of retries> <command> [ <command arg> ... ]"
fi

N_ATTEMPTS="$1"
shift

if ! [ "$N_ATTEMPTS" -gt 0 ] 2>/dev/null; then
    die "Error: '$N_ATTEMPTS' is not a valid number of attempts, please provide a positive natural number."
fi

CMD=$*

for i in $(seq 1 "$N_ATTEMPTS"); do
    eecho "** Executing '$CMD' (attempt $i of $N_ATTEMPTS) **"
    eecho
    $CMD
    ret=$?
    eecho
    if [ $ret -eq 0 ]; then
        exit 0
    fi
    if [ "$i" -lt "$N_ATTEMPTS" ]; then
        sleep $DELAY_BETWEEN_ATTEMPTS
    fi
done

eecho
die "** Command failed $N_ATTEMPTS times in a row, giving up"
