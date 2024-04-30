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

import { DynamicTableLabel } from 'Components/DynamicIcon';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getTableUIState } from 'utils/getTableUIState';
import { getUrlQueryStringForSearchFilter, getHasSearchApplied } from 'utils/searchUtils';

import { parseWorkloadQuerySearchFilter } from '../../utils/searchUtils';

import CVEsTable from './CVEsTable';
import useNodeVulnerabilities from './useNodeVulnerabilities';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';

export type NodePageVulnerabilitiesProps = {
    nodeId: string;
};

function NodePageVulnerabilities({ nodeId }: NodePageVulnerabilitiesProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);
    const query = getUrlQueryStringForSearchFilter(querySearchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    const { data, loading, error } = useNodeVulnerabilities(nodeId, query, page, perPage);

    const tableState = getTableUIState({
        isLoading: loading,
        error,
        data: data?.node.nodeVulnerabilities,
        searchFilter: querySearchFilter,
    });

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>Review and triage vulnerability data scanned on this node</Text>
            </PageSection>
            <PageSection isFilled className="pf-v5-u-display-flex pf-v5-u-flex-direction-column">
                <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100 pf-v5-u-p-lg">
                    <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                        <SplitItem isFilled>
                            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                <Title headingLevel="h2" className="pf-v5-u-w-50">
                                    {data ? (
                                        `${pluralize(
                                            data.node.nodeVulnerabilityCount,
                                            'result'
                                        )} found`
                                    ) : (
                                        <Skeleton screenreaderText="Loading node vulnerability count" />
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

export default NodePageVulnerabilities;
