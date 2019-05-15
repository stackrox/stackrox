#! /bin/sh

runQuery()
{
    /influx -execute "$1"
}

statusCheck()
{
    runQuery "SHOW DATABASES" > /dev/null 2>&1
}

echo "Waiting for InfluxDB server to start"
while ! statusCheck; do
    echo -n "."
    sleep 1
done

# Create the databases
runQuery "CREATE DATABASE \"telegraf_12h\" WITH DURATION 12h NAME \"12_hours\""

echo "Successfully started InfluxDB"
