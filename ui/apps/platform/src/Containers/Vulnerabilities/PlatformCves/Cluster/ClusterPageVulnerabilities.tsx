import React from 'react';
import {
    Flex,
    PageSection,
    Skeleton,
    Split,
    SplitItem,
    Text,
    Title,
    pluralize,
} from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import { getHasSearchApplied, getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { getTableUIState } from 'utils/getTableUIState';

import { DynamicTableLabel } from 'Components/DynamicIcon';
import { parseWorkloadQuerySearchFilter } from '../../utils/searchUtils';
import { DEFAULT_PAGE_SIZE } from '../../constants';

import useClusterVulnerabilities from './useClusterVulnerabilities';
import CVEsTable from './CVEsTable';

export type ClusterPageVulnerabilitiesProps = {
    clusterId: string;
};

function ClusterPageVulnerabilities({ clusterId }: ClusterPageVulnerabilitiesProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);
    const query = getUrlQueryStringForSearchFilter(querySearchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage } = useURLPagination(DEFAULT_PAGE_SIZE);

    const { data, loading, error } = useClusterVulnerabilities(clusterId, query, page, perPage);

    const tableState = getTableUIState({
        isLoading: loading,
        error,
        data: data?.cluster.clusterVulnerabilities,
        searchFilter: querySearchFilter,
    });

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>Review and triage vulnerability data scanned on this cluster</Text>
            </PageSection>
            <PageSection isFilled className="pf-v5-u-display-flex pf-v5-u-flex-direction-column">
                <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100 pf-v5-u-p-lg">
                    <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                        <SplitItem isFilled>
                            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                <Title headingLevel="h2" className="pf-v5-u-w-50">
                                    {data ? (
                                        `${pluralize(
                                            data.cluster.clusterVulnerabilityCount,
                                            'result'
                                        )} found`
                                    ) : (
                                        <Skeleton screenreaderText="Loading cluster vulnerability count" />
                                    )}
                                </Title>
                                {isFiltered && <DynamicTableLabel />}
                            </Flex>
                        </SplitItem>
                    </Split>
                    <CVEsTable tableState={tableState} />
                </div>
            </PageSection>
        </>
    );
}

export default ClusterPageVulnerabilities;
