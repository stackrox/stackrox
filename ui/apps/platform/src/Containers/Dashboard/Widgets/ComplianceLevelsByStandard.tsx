import React, { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import {
    Flex,
    FlexItem,
    Title,
    FormGroup,
    ToggleGroup,
    ToggleGroupItem,
    Button,
    Form,
    EmptyState,
    EmptyStateIcon,
    EmptyStateVariant,
    EmptyStateBody,
} from '@patternfly/react-core';
import { SyncIcon } from '@patternfly/react-icons';
import { useQuery } from '@apollo/client';
import isEqual from 'lodash/isEqual';
import sortBy from 'lodash/sortBy';

import LinkShim from 'Components/PatternFly/LinkShim';
import WidgetCard from 'Components/PatternFly/WidgetCard';
import useURLSearch from 'hooks/useURLSearch';
import useWidgetConfig from 'hooks/useWidgetConfig';
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
import WidgetOptionsMenu from './WidgetOptionsMenu';
import WidgetOptionsResetButton from './WidgetOptionsResetButton';

const fieldIdPrefix = 'compliance-levels-by-standard';

function ComplianceScanEmptyState() {
    return (
        <EmptyState className="pf-u-h-100" variant={EmptyStateVariant.xs}>
            <EmptyStateIcon className="pf-u-font-size-xl" icon={SyncIcon} />
            <Title headingLevel="h3" size="md">
                No Standard results available.
            </Title>
            <EmptyStateBody>Run a scan on the Compliance page.</EmptyStateBody>
            <Button component={LinkShim} href={complianceBasePath}>
                Go to compliance
            </Button>
        </EmptyState>
    );
}

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

    if (sortDirection === 'desc') {
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

const defaultConfig = { sortDataBy: 'asc' } as const;

function ComplianceLevelsByStandard() {
    const { searchFilter } = useURLSearch();
    const { pathname } = useLocation();
    const [{ sortDataBy }, updateConfig] = useWidgetConfig<{ sortDataBy: SortBy }>(
        'ComplianceLevelsByStandard',
        pathname,
        defaultConfig
    );

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
        () =>
            processData(searchFilter, sortDataBy, data || previousData)
                ?.slice(0, 6)
                ?.reverse(), // Reverse since Victory charts renders items from bottom to top
        [searchFilter, sortDataBy, data, previousData]
    );

    const isOptionsChanged = !isEqual(sortDataBy, defaultConfig.sortDataBy);

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
                        {isOptionsChanged && (
                            <WidgetOptionsResetButton onClick={() => updateConfig(defaultConfig)} />
                        )}
                        <WidgetOptionsMenu
                            bodyContent={
                                <Form>
                                    <FormGroup fieldId={`${fieldIdPrefix}-sort-by`} label="Sort by">
                                        <ToggleGroup aria-label="Sort coverage by ascending or descending percentage">
                                            <ToggleGroupItem
                                                text="Ascending"
                                                buttonId={`${fieldIdPrefix}-sort-by-asc`}
                                                isSelected={sortDataBy === 'asc'}
                                                onChange={() => updateConfig({ sortDataBy: 'asc' })}
                                            />
                                            <ToggleGroupItem
                                                text="Descending"
                                                buttonId={`${fieldIdPrefix}-sort-by-desc`}
                                                isSelected={sortDataBy === 'desc'}
                                                onChange={() =>
                                                    updateConfig({ sortDataBy: 'desc' })
                                                }
                                            />
                                        </ToggleGroup>
                                    </FormGroup>
                                </Form>
                            }
                        />
                        <Button variant="secondary" component={LinkShim} href={complianceBasePath}>
                            View all
                        </Button>
                    </FlexItem>
                </Flex>
            }
        >
            {complianceData && complianceData.length > 0 ? (
                <ComplianceLevelsByStandardChart complianceData={complianceData} />
            ) : (
                <ComplianceScanEmptyState />
            )}
        </WidgetCard>
    );
}

export default ComplianceLevelsByStandard;
