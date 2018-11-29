#! /bin/sh

# let influx get to a boot up point. Should be a query to see if its alive with a max timeout
sleep 10

runQuery()
{
    /influx -execute "$1"
}

# Create the databases
runQuery "CREATE DATABASE \"telegraf_12h\" WITH DURATION 12h NAME \"12_hours\""
runQuery "CREATE DATABASE \"telegraf_2w\" WITH DURATION 2w NAME \"2_weeks\""
runQuery "CREATE DATABASE \"telegraf_forever\""

# Create the continuous queries for down sampling
runQuery "CREATE CONTINUOUS QUERY \"telegraf_downsample_1m\" ON \"telegraf_12h\" BEGIN SELECT min(*), max(*), mean(*) INTO \"telegraf_2w\".\"2_weeks\".:MEASUREMENT FROM /.*/ GROUP BY time(1m),* END"
runQuery "CREATE CONTINUOUS QUERY \"telegraf_downsample_10m\" ON \"telegraf_12h\" BEGIN SELECT min(*), max(*), mean(*) INTO \"telegraf_forever\".\"autogen\".:MEASUREMENT FROM /.*/ GROUP BY time(10m),* END"
