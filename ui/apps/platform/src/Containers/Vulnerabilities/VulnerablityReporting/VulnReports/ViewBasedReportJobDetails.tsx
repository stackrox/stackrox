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
import { getSearchFilterFromSearchString, searchValueAsArray } from 'utils/searchUtils';
import { isVulnerabilitySeverity } from 'types/cve.proto';
import { formatCveDiscoveredTime } from '../../utils/vulnerabilityUtils';

export type ViewBasedReportJobDetailsProps = {
    reportSnapshot: ViewBasedReportSnapshot;
};

function ViewBasedReportJobDetails({ reportSnapshot }: ViewBasedReportJobDetailsProps) {
    const query = getSearchFilterFromSearchString(reportSnapshot.viewBasedVulnReportFilters.query);

    // Extract vulnerability-specific filters
    const findValueByKey = (
        obj: Partial<Record<string, string | string[]>>,
        targetKey: string
    ): string | string[] | undefined => {
        const foundKey = Object.keys(obj).find(
            (key) => key.toLowerCase() === targetKey.toLowerCase()
        );
        return foundKey ? obj[foundKey] : undefined;
    };

    const severityValues = findValueByKey(query, 'Severity');
    const cveDiscoveredTimeValues = findValueByKey(query, 'CVE Created Time');

    const validSeverities = severityValues
        ? searchValueAsArray(severityValues).filter((severity) => isVulnerabilitySeverity(severity))
        : [];

    const validCveDiscoveredTimes = cveDiscoveredTimeValues
        ? searchValueAsArray(cveDiscoveredTimeValues)
        : [];

    // Create scope filters excluding vulnerability-specific ones
    const scopeFilters = Object.fromEntries(
        Object.entries(query).filter(
            ([key]) => key.toLowerCase() !== 'severity' && key.toLowerCase() !== 'cve created time'
        )
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
                    <DescriptionListDescription>{reportSnapshot.name}</DescriptionListDescription>
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
                        {validSeverities.length > 0 ? (
                            <Stack>
                                {validSeverities.map((severity) => (
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
                        {validCveDiscoveredTimes.length > 0 ? (
                            <Stack>
                                {validCveDiscoveredTimes.map((timeValue) => (
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
