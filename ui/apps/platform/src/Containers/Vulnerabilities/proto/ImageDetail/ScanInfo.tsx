import {
    Card,
    CardBody,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
} from '@patternfly/react-core';

type ScanInfoProps = {
    scannerVersion?: string;
    bundleVersion?: string;
    dataSources?: string[];
    scanTime?: string;
};

/**
 * Standalone card showing scanner metadata for an image scan.
 * This is a prototype component, not wired into the existing image detail page.
 */
function ScanInfo({
    scannerVersion = 'N/A',
    bundleVersion = 'N/A',
    dataSources = [],
    scanTime,
}: ScanInfoProps) {
    const formattedScanTime = scanTime
        ? new Date(scanTime).toLocaleString()
        : 'N/A';

    return (
        <Card>
            <CardTitle>Scan Information</CardTitle>
            <CardBody>
                <DescriptionList isHorizontal>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Scanner Version</DescriptionListTerm>
                        <DescriptionListDescription>{scannerVersion}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Bundle Version</DescriptionListTerm>
                        <DescriptionListDescription>{bundleVersion}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Data Sources</DescriptionListTerm>
                        <DescriptionListDescription>
                            {dataSources.length > 0 ? dataSources.join(', ') : 'N/A'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Scan Time</DescriptionListTerm>
                        <DescriptionListDescription>{formattedScanTime}</DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
            </CardBody>
        </Card>
    );
}

export default ScanInfo;
