#!/bin/bash

function test_exec {
    exec_name=$1
    cp /bin/echo /usr/local/bin/${exec_name}
    ${exec_name} hello
}

test_exec sshd

sleep 36000
