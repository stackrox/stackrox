import React from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    Stack,
    Title,
} from '@patternfly/react-core';

import { OnDemandReportSnapshot } from 'services/ReportsService.types';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';

export type OnDemandReportJobDetailsProps = {
    reportSnapshot: OnDemandReportSnapshot;
};

function OnDemandReportJobDetails({ reportSnapshot }: OnDemandReportJobDetailsProps) {
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            <Title headingLevel="h2">Report details</Title>
            <DescriptionList
                columnModifier={{
                    default: '3Col',
                }}
            >
                <DescriptionListGroup>
                    <DescriptionListTerm>Name</DescriptionListTerm>
                    <DescriptionListDescription>
                        {reportSnapshot.requestName}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Results type</DescriptionListTerm>
                    <DescriptionListDescription>Vulnerabilities</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Area of concern</DescriptionListTerm>
                    <DescriptionListDescription>
                        {reportSnapshot.areaOfConcern}
                    </DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
            <Title headingLevel="h2">Scope details</Title>
            <DescriptionList
                columnModifier={{
                    default: '1Col',
                }}
            >
                <DescriptionListGroup>
                    <DescriptionListTerm>Scoping method</DescriptionListTerm>
                    <DescriptionListDescription>Using filters</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Scope filters</DescriptionListTerm>
                    <DescriptionListDescription>
                        {reportSnapshot.vulnReportFilters.query}
                    </DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
            <Title headingLevel="h2">Vulnerability parameters</Title>
            <DescriptionList
                columnModifier={{
                    default: '3Col',
                }}
            >
                <DescriptionListGroup>
                    <DescriptionListTerm>CVE severity</DescriptionListTerm>
                    <DescriptionListDescription>
                        <Stack>
                            <VulnerabilitySeverityIconText severity="CRITICAL_VULNERABILITY_SEVERITY" />
                            <VulnerabilitySeverityIconText severity="IMPORTANT_VULNERABILITY_SEVERITY" />
                        </Stack>
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>CVEs discovered since</DescriptionListTerm>
                    <DescriptionListDescription>All time</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Optional columns</DescriptionListTerm>
                    <DescriptionListDescription>
                        <Stack>
                            {reportSnapshot.vulnReportFilters.includeNvdCvss && <div>NVD CVSS</div>}
                            {reportSnapshot.vulnReportFilters.includeEpssProbability && (
                                <div>EPSS probability</div>
                            )}
                        </Stack>
                    </DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
        </Flex>
    );
}

export default OnDemandReportJobDetails;
