#!/usr/bin/env bash

# Report how many alerts a file-activity test policy generated.
# The policy name differs per workload (the berserker test uses a distinct
# policy from the fake workload), so it is passed in; callers must use the same
# name they created the policy with. Defaults to the fake-workload policy name.
policy_name="${1:-File Activity Test Policy}"

pass="$(kubectl -n stackrox get secret central-db-password -o json | jq .data.password --raw-output | base64 --decode)"

kubectl -n stackrox port-forward svc/central-db 8080:5432 > /dev/null 2>&1 &
pid=$!
sleep 5

PGPASSWORD="$pass" psql -U postgres -d central_active -h 127.0.0.1 -p 8080 -v policy_name="$policy_name" << 'EOL'
      -- Count alerts triggered by file activity policy
      select count(*) as total_file_activity_alerts from alerts where policy_name = :'policy_name';

      -- Show breakdown by severity if any alerts exist
      select severity, count(*) as count from alerts
      where policy_name = :'policy_name'
      group by severity;
EOL

kill -9 "$pid" > /dev/null 2>&1
