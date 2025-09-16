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

import type { ViewBasedReportSnapshot } from 'services/ReportsService.types';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import { getSearchFilterFromSearchString } from 'utils/searchUtils';
import { isVulnerabilitySeverity, type VulnerabilitySeverity } from 'types/cve.proto';
import { getDate } from 'utils/dateUtils';

export type ViewBasedReportJobDetailsProps = {
    reportSnapshot: ViewBasedReportSnapshot;
};

function formatCveDiscoveredTime(value: string): string {
    try {
        // Parse the condition prefix and date
        const match = value.match(/^([<>]?)(.+)$/);
        if (!match) {
            return value; // Return original if parsing fails
        }

        const [, condition, dateStr] = match;
        const date = new Date(dateStr);

        // Check if date is valid
        if (Number.isNaN(date.getTime())) {
            return value; // Return original if date is invalid
        }

        const formattedDate = getDate(date);

        // Map conditions to user-friendly text
        switch (condition) {
            case '>':
                return `After ${formattedDate}`;
            case '<':
                return `Before ${formattedDate}`;
            default:
                return `On ${formattedDate}`;
        }
    } catch {
        // Return original value if any error occurs
        return value;
    }
}

function ViewBasedReportJobDetails({ reportSnapshot }: ViewBasedReportJobDetailsProps) {
    const query = getSearchFilterFromSearchString(reportSnapshot.viewBasedVulnReportFilters.query);

    // Extract vulnerability-specific filters
    const severityValues = query.Severity;
    const cveDiscoveredTimeValues = query['CVE Discovered Time'];

    // Create scope filters excluding vulnerability-specific ones
    const scopeFilters = Object.fromEntries(
        Object.entries(query).filter(([key]) => key !== 'Severity' && key !== 'CVE Discovered Time')
    );

    const scopeFilterChips = Object.entries(scopeFilters).map(([key, value]) => {
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
                        {severityValues ? (
                            <Stack>
                                {(Array.isArray(severityValues) ? severityValues : [severityValues])
                                    .filter((severity): severity is VulnerabilitySeverity =>
                                        isVulnerabilitySeverity(severity)
                                    )
                                    .map((severity) => (
                                        <VulnerabilitySeverityIconText
                                            key={severity}
                                            severity={severity}
                                        />
                                    ))}
                            </Stack>
                        ) : (
                            'All severities'
                        )}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>CVEs discovered time</DescriptionListTerm>
                    <DescriptionListDescription>
                        {cveDiscoveredTimeValues ? (
                            <Stack>
                                {(Array.isArray(cveDiscoveredTimeValues)
                                    ? cveDiscoveredTimeValues
                                    : [cveDiscoveredTimeValues]
                                ).map((timeValue) => (
                                    <div key={timeValue}>{formatCveDiscoveredTime(timeValue)}</div>
                                ))}
                            </Stack>
                        ) : (
                            'All time'
                        )}
                    </DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
        </Flex>
    );
}

export default ViewBasedReportJobDetails;
