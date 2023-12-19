#!/bin/bash

export GOLANG_PROTOBUF_REGISTRATION_CONFLICT=ignore
generated="generated/storage"

for value in vtproto gogo csproto
do
   echo $value
   cd $value/${generated}
   go test -run=. -bench=. -benchmem -count 10 -benchtime 3s ./...
   cd -
done
