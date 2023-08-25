#!/usr/bin/env bash

got_term() {
    echo $$ > "$TEST_TERM_PIDFILE"
}

if [[ -n "$TEST_PIDFILE" ]]; then
    echo $$ > "$TEST_PIDFILE"
fi

if [[ -n "$TEST_TERM_PIDFILE" ]]; then
    trap got_term SIGTERM
fi

sleep 100 &
wait $!
