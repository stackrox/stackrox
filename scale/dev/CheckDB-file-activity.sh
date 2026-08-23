#!/usr/bin/env bash

pass="$(kubectl -n stackrox get secret central-db-password -o json | jq .data.password --raw-output | base64 --decode)"

kubectl -n stackrox port-forward svc/central-db 8080:5432 > /dev/null 2>&1 &
pid=$!
sleep 5

PGPASSWORD="$pass" psql -U postgres -d central_active -h 127.0.0.1 -p 8080 << EOL
      -- Count alerts triggered by file activity policy
      select count(*) as total_file_activity_alerts from alerts where policy_name = 'File Activity Test Policy';

      -- Show breakdown by severity if any alerts exist
      select severity, count(*) as count from alerts
      where policy_name = 'File Activity Test Policy'
      group by severity;
EOL

kill -9 "$pid" > /dev/null 2>&1
