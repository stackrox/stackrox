#!/bin/bash
set -eu

delete_pods_of_deployment() {
    local deployment_name=$1

    case "${deployment_name}" in
        central|sensor|collector) : ;;
        *)
          echo "Unsupported deployment name ${deployment_name}"
          return 1
          ;;
    esac

    if (( RANDOM % 2 )); then
        echo "Skipping kill of ${deployment_name}";
        return 0;
    fi

    for pod in $(kubectl -n stackrox get po | grep "${deployment_name}" | grep -v -e 'Terminating' -e 'ContainerCreating' -e 'Error' -e 'Completed' -e 'CrashLoopBackOff' -e '1/2' | tee temp.out | awk '{print $1}'); do
        echo "Killing pod ${pod}"
        top -b -n 1 -o %MEM
        cat temp.out
        case "${deployment_name}" in
            central|sensor)
                kubectl -n stackrox exec "${pod}" -c "${deployment_name}" -- kill 1
                ;;
            collector)
                # This is necessary because kubectl exec doesn't handle string quoting properly, so sh -c "kill 1" doesn't work.
                # We can't directly use "kill 1" because kill is a shell builtin on debian, not a binary.
                echo kill 1 | kubectl -n stackrox exec -i "${pod}" -c "${deployment_name}" -- sh -
                ;;
        esac
    done
}

main() {
    echo "Chaos monkeying..."
    for i in $(seq 1 10);
    do
        echo "Deletion round ${i}"
        delete_pods_of_deployment central
        sleep 5
        delete_pods_of_deployment sensor
        sleep 5
        delete_pods_of_deployment collector
        sleep 10
    done
}

main
