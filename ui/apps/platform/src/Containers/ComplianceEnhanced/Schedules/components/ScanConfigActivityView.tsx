import type { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    Title,
} from '@patternfly/react-core';

import type { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';

import { getTimeWithHourMinuteFromISO8601 } from '../compliance.scanConfigs.utils';

type ScanConfigActivityViewProps = {
    headingLevel: 'h2' | 'h3';
    scanConfig: ComplianceScanConfigurationStatus;
};

function ScanConfigActivityView({
    headingLevel,
    scanConfig,
}: ScanConfigActivityViewProps): ReactElement {
    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel={headingLevel}>Activity</Title>
            <DescriptionList isCompact isHorizontal>
                <DescriptionListGroup>
                    <DescriptionListTerm>Last scanned</DescriptionListTerm>
                    <DescriptionListDescription>
                        {scanConfig.lastExecutedTime
                            ? getTimeWithHourMinuteFromISO8601(scanConfig.lastExecutedTime)
                            : 'Scan is in progress'}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Last updated</DescriptionListTerm>
                    <DescriptionListDescription>
                        {getTimeWithHourMinuteFromISO8601(scanConfig.lastUpdatedTime)}
                    </DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
        </Flex>
    );
}

export default ScanConfigActivityView;
