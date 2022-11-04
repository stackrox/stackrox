# Instructions for DGM testing for moss

dgm_moss_test.go is a moss component level test that exercises moss.

Included in the test is a simulator of the moss herder memory stalling
functionality.

The test can be configured to generate a Results*.json file that can be
used with python script graph/dgm-moss-ploy.py to plot the resutls.

## Steps

Here is an example shell script to exercise the test.

```
#!/bin/bash
findDisk()
{
    dev=`df ${MossStore} | grep -v Filesystem | awk {'print $1'}`
    dirName=`dirname ${dev}`
    devName=`basename ${dev} ${dirName}`
    echo ${devName}
}

clearCaches()
{
    sudo sync
    sudo sh -c "echo 1 > /proc/sys/vm/drop_caches"
    sudo sh -c "echo 2 > /proc/sys/vm/drop_caches"
    sudo sh -c "echo 3 > /proc/sys/vm/drop_caches"

}

MossStore="/mossstore/MossStoreTest"
mkdir -p -m=0777 ${MossStore}
diskMonitor=`findDisk`
runDesc="${gitInfo}"

go test -v -test.run TestMossDGM -test.timeout 99999s -runTime 5m -memQuota 4gb -keyLength 48 -valueLength 48 -keyOrder random -numWriters 1 -writeBatchSize 100000 -writeBatchThinkTime 1ms -numReaders 16 -readBatchSize 100 -readBatchThinkTime 1ms -outputToFile -dbPath ${MossStore} -diskMonitor ${diskMonitor} -runDescription ${runDesc} -dbCreate
```

## Ploting a graph of the results

The python script requires numpy, pandas, matploglib

```
python dgm-moss-plot.py Results_...json
```

This will process a Results...png file.

## Notes

This test only runs properly on Linux as it gathers stats from /proc.
