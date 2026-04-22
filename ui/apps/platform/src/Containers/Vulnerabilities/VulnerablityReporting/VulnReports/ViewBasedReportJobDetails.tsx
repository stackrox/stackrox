import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    Title,
} from '@patternfly/react-core';

import type { ViewBasedReportSnapshot } from 'services/ReportsService.types';
import CompoundSearchFilterDescriptionListGroups from 'Components/CompoundSearchFilter/components/CompoundSearchFilterDescriptionListGroups';
import { getSearchFilterFromSearchString } from 'utils/searchUtils';
import {
    attributesSeparateFromConfigForViewBasedReport,
    configForViewBasedReport,
} from '../../searchFilterConfig';

export type ViewBasedReportJobDetailsProps = {
    reportSnapshot: ViewBasedReportSnapshot;
};

function ViewBasedReportJobDetails({ reportSnapshot }: ViewBasedReportJobDetailsProps) {
    const { name, viewBasedVulnReportFilters } = reportSnapshot;
    const { query } = viewBasedVulnReportFilters;

    const searchFilter = getSearchFilterFromSearchString(query);

    // Render separate attributes (more likely to be specified) preceding config.
    const attributesFromConfig = configForViewBasedReport.flatMap(({ attributes }) => attributes);
    const attributes = [...attributesSeparateFromConfigForViewBasedReport, ...attributesFromConfig];

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            <Title headingLevel="h2">Details</Title>
            <DescriptionList
                isCompact
                isHorizontal
                horizontalTermWidthModifier={{ default: '20ch' }}
            >
                <DescriptionListGroup>
                    <DescriptionListTerm>Name</DescriptionListTerm>
                    <DescriptionListDescription>{name}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Report type</DescriptionListTerm>
                    <DescriptionListDescription>Image vulnerabilities</DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
            <Title headingLevel="h2">Filters</Title>
            <DescriptionList
                isCompact
                isHorizontal
                horizontalTermWidthModifier={{ default: '20ch' }}
            >
                <CompoundSearchFilterDescriptionListGroups
                    attributes={attributes}
                    searchFilter={searchFilter}
                />
            </DescriptionList>
        </Flex>
    );
}

export default ViewBasedReportJobDetails;
