import React from 'react';
import {
    Chip,
    ChipGroup,
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
import { getSearchFilterFromSearchString } from 'utils/searchUtils';

export type OnDemandReportJobDetailsProps = {
    reportSnapshot: OnDemandReportSnapshot;
};

function OnDemandReportJobDetails({ reportSnapshot }: OnDemandReportJobDetailsProps) {
    // @TODO: We need to separate the "CVE Severity" and "CVEs discovered since" filters from the rest of the filters.
    // The relevant search terms are called "Severity" and "CVE Discovered Time".
    const query = getSearchFilterFromSearchString(reportSnapshot.vulnReportFilters.query);
    const scopeFilterChips = Object.entries(query).map(([key, value]) => {
        if (!value) {
            return null;
        }
        if (typeof value === 'string') {
            return (
                <ChipGroup categoryName={key}>
                    <Chip key={value} isReadOnly>
                        {value}
                    </Chip>
                </ChipGroup>
            );
        }
        return (
            <ChipGroup categoryName={key}>
                {value.map((currentChip) => (
                    <Chip key={currentChip} isReadOnly>
                        {currentChip}
                    </Chip>
                ))}
            </ChipGroup>
        );
    });

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
                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>{scopeFilterChips}</Flex>
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
