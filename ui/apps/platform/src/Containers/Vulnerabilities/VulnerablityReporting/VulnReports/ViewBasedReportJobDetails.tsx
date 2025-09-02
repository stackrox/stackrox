import React from 'react';
import type { ReactElement } from 'react';
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

import useFeatureFlags from 'hooks/useFeatureFlags';
import type { ViewBasedReportSnapshot } from 'services/ReportsService.types';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import { getSearchFilterFromSearchString } from 'utils/searchUtils';

export type ViewBasedReportJobDetailsProps = {
    reportSnapshot: ViewBasedReportSnapshot;
};

function ViewBasedReportJobDetails({ reportSnapshot }: ViewBasedReportJobDetailsProps) {
    // TODO Analyze pro and con of redundancy with ReportParametersDedtails component.
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const optionalColumnsDescriptions: ReactElement[] = [];
    if (isFeatureFlagEnabled('ROX_SCANNER_V4') && reportSnapshot.vulnReportFilters.includeNvdCvss) {
        optionalColumnsDescriptions.push(
            <DescriptionListDescription key="includeNvdCvss">NVDCVSS</DescriptionListDescription>
        );
    }
    if (
        isFeatureFlagEnabled('ROX_SCANNER_V4') &&
        reportSnapshot.vulnReportFilters.includeEpssProbability
    ) {
        optionalColumnsDescriptions.push(
            <DescriptionListDescription key="includeEpssProbability">
                EPSS Probability Percentage
            </DescriptionListDescription>
        );
    }
    /*
    if (
        isFeatureFlagEnabled('ROX_SCANNER_V4') &&
        reportSnapshot.vulnReportFilters.includeAdvisory
    ) {
        optionalColumnsDescriptions.push(
            <DescriptionListDescription key="includeAdvisory">
                Advisory Name and Advisory Link
            </DescriptionListDescription>
        );
    }
    */
    /*
    // Ross CISA KEV includeKnownExploit?
    // Probably for 4.9 because optional columns might not be up to date for view-based reports.
    if (
        isFeatureFlagEnabled('ROX_SCANNER_V4') &&
        isFeatureFlagEnabled('ROX_WHATEVER') &&
        formValues.reportParameters.includeKnownExploit
    ) {
        optionalColumnsDescriptions.push(
            <DescriptionListDescription key="includeKnownExploit">
                Known exploit
            </DescriptionListDescription>
        );
    }
    */

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
                {optionalColumnsDescriptions.length !== 0 && (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Optional columns</DescriptionListTerm>
                        {optionalColumnsDescriptions}
                    </DescriptionListGroup>
                )}
            </DescriptionList>
        </Flex>
    );
}

export default ViewBasedReportJobDetails;
