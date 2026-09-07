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
import { Origin } from 'Components/CompoundSearchFilter/attributes/imageCVE';
import { getSearchFilterFromSearchString } from 'utils/searchUtils';
import {
    attributesSeparateFromConfigForImageVulnerabilityReport,
    searchFilterConfigForWorkloadVulnerabilityResultsAndViewBasedReport,
} from '../../searchFilterConfig';

export type ViewBasedReportJobDetailsProps = {
    reportSnapshot: ViewBasedReportSnapshot;
};

function ViewBasedReportJobDetails({ reportSnapshot }: ViewBasedReportJobDetailsProps) {
    const { name, viewBasedVulnReportFilters } = reportSnapshot;
    const { query } = viewBasedVulnReportFilters;

    const searchFilter = getSearchFilterFromSearchString(query);

    // Render separate attributes (more likely to be specified) preceding config.
    const attributesFromConfig =
        searchFilterConfigForWorkloadVulnerabilityResultsAndViewBasedReport.flatMap(
            ({ attributes }) => attributes
        );
    const attributes = [
        ...attributesSeparateFromConfigForImageVulnerabilityReport,
        ...attributesFromConfig,
        // Origin is not part of the shared workload/view-based config (that config also drives
        // the overview page filter input, which is out of scope). Append it here so a view-based
        // report created from the image/deployment single pages displays the CVE origin filter.
        Origin,
    ];

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
