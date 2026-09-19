#!/usr/bin/env bash

pass="$(kubectl -n stackrox get secret central-db-password -o json | jq .data.password --raw-output | base64 --decode)"

kubectl -n stackrox port-forward svc/central-db 8080:5432 > /dev/null 2>&1 &
pid=$!
sleep 5

PGPASSWORD="$pass" psql -U postgres -d central_active -h 127.0.0.1 -p 8080 << EOL
      select count(*) from alerts where policy_name = 'Unauthorized Process Execution' ;
EOL

kill -9 "$pid" > /dev/null 2>&1

