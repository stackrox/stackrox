import React from 'react';
import {
    Card,
    CardBody,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Timestamp,
} from '@patternfly/react-core';

import { ComplianceScanConfigurationStatus } from 'services/ComplianceEnhancedService';
import { formatScanSchedule } from '../compliance.scanConfigs.utils';

type ScanConfigParametersProps = {
    scanConfig: ComplianceScanConfigurationStatus;
};

function ScanConfigParameters({ scanConfig }: ScanConfigParametersProps): React.ReactElement {
    return (
        <Card className="pf-u-h-100">
            <CardTitle component="h2">Parameters</CardTitle>
            <CardBody>
                <DescriptionList>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Name</DescriptionListTerm>
                        <DescriptionListDescription>
                            {scanConfig.scanName}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Description</DescriptionListTerm>
                        <DescriptionListDescription>
                            {scanConfig.scanConfig.description || <em>No description</em>}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Schedule</DescriptionListTerm>
                        <DescriptionListDescription>
                            {formatScanSchedule(scanConfig.scanConfig.scanSchedule)}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Last run</DescriptionListTerm>
                        <DescriptionListDescription>
                            {/* TODO: (vjw, 2024-01-16) using the `lastUpdatedTime` field
                                      as a placeholder for now
                                      There is a comment in GitHub that the correct field,
                                      `lastRun` will be available soon.
                            */}
                            <Timestamp
                                date={new Date(scanConfig.lastUpdatedTime)}
                                dateFormat="short"
                                timeFormat="long"
                                className="pf-u-color-100 pf-u-font-size-md"
                            />
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
            </CardBody>
        </Card>
    );
}

export default ScanConfigParameters;
