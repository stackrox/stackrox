import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    Title,
} from '@patternfly/react-core';

import type { ViewBasedReportSnapshot } from 'services/ReportsService.types';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import { getSearchFilterFromSearchString } from 'utils/searchUtils';
import {
    attributesSeparateFromConfigForViewBasedReport,
    configForViewBasedReport,
} from '../../searchFilterConfig';

export type ViewBasedReportJobDetailsProps = {
    reportSnapshot: ViewBasedReportSnapshot;
};

function ViewBasedReportJobDetails({ reportSnapshot }: ViewBasedReportJobDetailsProps) {
    const query = getSearchFilterFromSearchString(reportSnapshot.viewBasedVulnReportFilters.query);

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
        </Flex>
    );
}

export default ViewBasedReportJobDetails;
