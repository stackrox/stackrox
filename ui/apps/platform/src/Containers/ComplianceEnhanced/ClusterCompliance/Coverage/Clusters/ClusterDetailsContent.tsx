import React, { useEffect, useState } from 'react';
import { Alert, Divider, Flex, FlexItem, Gallery, GalleryItem } from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import { ComplianceClusterScanStats } from 'services/ComplianceEnhancedService';

import RadioButtonWithStats from '../Components/RadioButtonWithStats';

export type ClusterDetailsContentProps = {
    scanRecords: ComplianceClusterScanStats[];
};

function ClusterDetailsContent({ scanRecords }: ClusterDetailsContentProps) {
    const scanNames = scanRecords.map((item) => item.scanStats.scanName) as [string, ...string[]];
    const [urlSelectedScan, setUrlSelectedScan] = useURLStringUnion('selectedScan', scanNames);
    const [selectedScan, setSelectedScan] = useState('');

    useEffect(() => {
        if (
            urlSelectedScan &&
            scanRecords.some((record) => record.scanStats.scanName === urlSelectedScan)
        ) {
            setSelectedScan(urlSelectedScan);
        } else if (scanRecords.length > 0) {
            setSelectedScan(scanRecords[0].scanStats.scanName);
        }
    }, [urlSelectedScan, scanRecords]);

    const handleSelectedScan = (scan) => {
        setUrlSelectedScan(scan);
        setSelectedScan(scan);
    };

    return (
        <Flex
            direction={{ default: 'column' }}
            className="pf-u-background-color-100 pf-u-p-lg"
            spaceItems={{ default: 'spaceItemsLg' }}
            flexWrap={{ default: 'nowrap' }}
        >
            <FlexItem>
                <Alert
                    variant="info"
                    title="View results by scan schedule today. Support for viewing compliance results by profiles is coming soon."
                    component="div"
                    isInline
                />
            </FlexItem>
            <FlexItem>
                <Gallery hasGutter>
                    {scanRecords.map(({ scanStats }) => (
                        <GalleryItem key={scanStats.scanName}>
                            <RadioButtonWithStats
                                key={scanStats.scanName}
                                scanStats={scanStats}
                                isSelected={scanStats.scanName === selectedScan}
                                onSelected={handleSelectedScan}
                            />
                        </GalleryItem>
                    ))}
                </Gallery>
            </FlexItem>
            <FlexItem>
                <Divider component="div" />
            </FlexItem>
        </Flex>
    );
}

export default ClusterDetailsContent;
