import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    // Stack,
    Title,
} from '@patternfly/react-core';

import type { ViewBasedReportSnapshot } from 'services/ReportsService.types';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
// import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import {
    getSearchFilterFromSearchString,
    // getValueByCaseInsensitiveKey,
    // searchValueAsArray,
} from 'utils/searchUtils';
// import { isVulnerabilitySeverity } from 'types/cve.proto';
// import { formatCveDiscoveredTime } from '../../utils/vulnerabilityUtils';
import {
    attributesSeparateFromConfigForViewBasedReport,
    configForViewBasedReport,
} from '../../searchFilterConfig';

export type ViewBasedReportJobDetailsProps = {
    reportSnapshot: ViewBasedReportSnapshot;
};

function ViewBasedReportJobDetails({ reportSnapshot }: ViewBasedReportJobDetailsProps) {
    const query = getSearchFilterFromSearchString(reportSnapshot.viewBasedVulnReportFilters.query);

    // Extract vulnerability-specific filters
    // const severityValues = getValueByCaseInsensitiveKey(query, 'Severity');
    // const cveDiscoveredTimeValues = getValueByCaseInsensitiveKey(query, 'CVE Created Time');

    // const validSeverities = severityValues
    //     ? searchValueAsArray(severityValues).filter((severity) => isVulnerabilitySeverity(severity))
    //     : [];

    // const validCveDiscoveredTimes = cveDiscoveredTimeValues
    //     ? searchValueAsArray(cveDiscoveredTimeValues)
    //     : [];

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
                        <CompoundSearchFilterLabels
                            attributesSeparateFromConfig={
                                attributesSeparateFromConfigForViewBasedReport
                            }
                            config={configForViewBasedReport}
                            searchFilter={query}
                        />
                    </DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
            {/*
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
            */}
        </Flex>
    );
}

export default ViewBasedReportJobDetails;
