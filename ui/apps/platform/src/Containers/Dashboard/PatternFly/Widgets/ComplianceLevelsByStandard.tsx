import React, { useMemo, useState } from 'react';
import {
    Flex,
    FlexItem,
    Title,
    Dropdown,
    DropdownToggle,
    FormGroup,
    ToggleGroup,
    ToggleGroupItem,
    Button,
    Form,
} from '@patternfly/react-core';
import { useQuery } from '@apollo/client';
import { sortBy } from 'lodash';

import LinkShim from 'Components/PatternFly/LinkShim';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useURLSearch from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';

import {
    getRequestQueryStringForSearchFilter,
    getUrlQueryStringForSearchFilter,
} from 'utils/searchUtils';
import entityTypes, {
    StandardEntityType,
    standardTypes,
    standardEntityTypes,
} from 'constants/entityTypes';
import { AGGREGATED_RESULTS_ACROSS_ENTITIES } from 'queries/controls';
import { complianceBasePath, urlEntityListTypes } from 'routePaths';
import { standardLabels } from 'messages/standards';
import ComplianceLevelsByStandardChart, { ComplianceData } from './ComplianceLevelsByStandardChart';
import WidgetCard from './WidgetCard';

const fieldIdPrefix = 'compliance-levels-by-standard';

// Adapted from `processData` function in the original DashboardCompliance.js code
function processData(
    searchFilter: SearchFilter,
    sortDirection: SortBy,
    data?: AggregationResult
): ComplianceData | undefined {
    if (!data) {
        return undefined;
    }
    const { complianceStandards } = data;
    const modifiedData = data.controls.results.map((result) => {
        const aggregationName = result?.aggregationKeys[0]?.id;
        const standard = complianceStandards.find((cs) => cs.id === aggregationName);
        const { numPassing, numFailing } = result;
        const standardQueryValue =
            standardLabels[standard?.id] || aggregationName || 'Unrecognized standard';
        const query = getUrlQueryStringForSearchFilter({
            ...searchFilter,
            Cluster: searchFilter.Cluster || '*',
            standard: standardQueryValue,
        });
        const link = `${complianceBasePath}/${
            urlEntityListTypes[standardEntityTypes.CONTROL]
        }?${query}`;
        const modifiedResult = {
            name: standard?.name || aggregationName || 'Unrecognized standard',
            passing: Math.round((numPassing / (numFailing + numPassing)) * 100) || 0,
            link,
        };
        return modifiedResult;
    });

    const sorted = sortBy(modifiedData, [(datum) => datum.passing]);

    if (sortDirection === 'asc') {
        sorted.reverse();
    }

    return sorted;
}

type AggregationResult = {
    controls: {
        results: {
            aggregationKeys: {
                id: keyof typeof standardTypes;
                scope: StandardEntityType;
            }[];
            numFailing: number;
            numPassing: number;
            numSkipped: number;
            unit: StandardEntityType;
        }[];
    };
    complianceStandards: {
        id: string;
        name: string;
    }[];
};

type SortBy = 'asc' | 'desc';

function ComplianceLevelsByStandard() {
    const { isOpen: isOptionsOpen, onToggle: toggleOptionsOpen } = useSelectToggle();
    const { searchFilter } = useURLSearch();
    const [sortDataBy, setSortDataBy] = useState<SortBy>('asc');

    const where = getRequestQueryStringForSearchFilter({
        // We always need to include some value for Cluster, otherwise aggregation will be performed at the namespace level
        ...searchFilter,
        Cluster: searchFilter.Cluster || '*',
    });
    const variables = {
        groupBy: [entityTypes.STANDARD],
        where,
    };
    const { loading, error, data, previousData } = useQuery<AggregationResult>(
        AGGREGATED_RESULTS_ACROSS_ENTITIES,
        { variables }
    );

    const complianceData = useMemo(
        () => processData(searchFilter, sortDataBy, data || previousData)?.slice(0, 6),
        [searchFilter, sortDataBy, data, previousData]
    );

    return (
        <WidgetCard
            isLoading={loading && !complianceData}
            error={error}
            header={
                <Flex direction={{ default: 'row' }}>
                    <FlexItem grow={{ default: 'grow' }}>
                        <Title headingLevel="h2">Compliance by standard</Title>
                    </FlexItem>
                    <FlexItem>
                        <Dropdown
                            className="pf-u-mr-sm"
                            toggle={
                                <DropdownToggle
                                    id={`${fieldIdPrefix}-options-toggle`}
                                    toggleVariant="secondary"
                                    onToggle={toggleOptionsOpen}
                                >
                                    Options
                                </DropdownToggle>
                            }
                            position="right"
                            isOpen={isOptionsOpen}
                        >
                            <Form className="pf-u-px-md pf-u-py-sm">
                                <FormGroup fieldId={`${fieldIdPrefix}-sort-by`} label="Sort by">
                                    <ToggleGroup aria-label="Sort coverage by ascending or descending percentage">
                                        <ToggleGroupItem
                                            text="Ascending"
                                            buttonId={`${fieldIdPrefix}-sort-by-asc`}
                                            isSelected={sortDataBy === 'asc'}
                                            onChange={() => setSortDataBy('asc')}
                                        />
                                        <ToggleGroupItem
                                            text="Descending"
                                            buttonId={`${fieldIdPrefix}-sort-by-desc`}
                                            isSelected={sortDataBy === 'desc'}
                                            onChange={() => setSortDataBy('desc')}
                                        />
                                    </ToggleGroup>
                                </FormGroup>
                            </Form>
                        </Dropdown>
                        <Button variant="secondary" component={LinkShim} href={complianceBasePath}>
                            View all
                        </Button>
                    </FlexItem>
                </Flex>
            }
        >
            {complianceData && <ComplianceLevelsByStandardChart complianceData={complianceData} />}
        </WidgetCard>
    );
}

export default ComplianceLevelsByStandard;
