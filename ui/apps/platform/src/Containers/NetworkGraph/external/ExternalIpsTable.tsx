import React, { useCallback } from 'react';
import {
    Button,
    Divider,
    Flex,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { InnerScrollContainer, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useMetadata from 'hooks/useMetadata';
import useRestQuery from 'hooks/useRestQuery';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';
import { getExternalIpsFlowsMetadata } from 'services/NetworkService';
import { getTableUIState } from 'utils/getTableUIState';
import { getVersionedDocs } from 'utils/versioning';
import {
    ExternalNetworkFlowsMetadataResponse,
    ExternalSourceNetworkEntityInfo,
} from 'types/networkFlow.proto';
import { SearchFilter } from 'types/search';

import IPMatchFilter from '../common/IPMatchFilter';
import { EXTERNAL_SOURCE_ADDRESS_QUERY } from '../NetworkGraph.constants';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

export type ExternalIpsTableProps = {
    scopeHierarchy: NetworkScopeHierarchy;
    onExternalIPSelect: (externalIP: string) => void;
    advancedFilters?: SearchFilter;
    urlSearchFiltering: UseUrlSearchReturn;
    urlPagination: UseURLPaginationResult;
};

function ExternalIpsTable({
    scopeHierarchy,
    onExternalIPSelect,
    advancedFilters,
    urlSearchFiltering,
    urlPagination,
}: ExternalIpsTableProps) {
    const { version } = useMetadata();
    const { page, perPage, setPage, setPerPage } = urlPagination;
    const { searchFilter, setSearchFilter } = urlSearchFiltering;
    const clusterId = scopeHierarchy.cluster.id;
    const { namespaces, deployments } = scopeHierarchy;

    const fetchExternalIpsFlowsMetadata =
        useCallback((): Promise<ExternalNetworkFlowsMetadataResponse> => {
            return getExternalIpsFlowsMetadata(clusterId, namespaces, deployments, {
                sortOption: {},
                page,
                perPage,
                advancedFilters: searchFilter,
            });
        }, [page, perPage, clusterId, deployments, namespaces, searchFilter]);

    const {
        data: externalIpsFlowsMetadata,
        isLoading,
        error,
    } = useRestQuery(fetchExternalIpsFlowsMetadata);

    const tableState = getTableUIState({
        isLoading,
        data: externalIpsFlowsMetadata?.entities,
        error,
        searchFilter,
    });

    return (
        <>
            <Toolbar className="pf-v5-u-pb-md pf-v5-u-pt-0">
                <ToolbarContent className="pf-v5-u-px-0">
                    <ToolbarItem className="pf-v5-u-w-100">
                        <IPMatchFilter
                            searchFilter={searchFilter}
                            setSearchFilter={setSearchFilter}
                        />
                    </ToolbarItem>
                    <ToolbarItem className="pf-v5-u-w-100">
                        <SearchFilterChips
                            searchFilter={searchFilter}
                            onFilterChange={setSearchFilter}
                            filterChipGroupDescriptors={[
                                {
                                    displayName: 'CIDR',
                                    searchFilterName: EXTERNAL_SOURCE_ADDRESS_QUERY,
                                },
                            ]}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider />
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                        <Pagination
                            itemCount={externalIpsFlowsMetadata?.totalEntities ?? 0}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                            isCompact
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <InnerScrollContainer>
                <Table variant="compact">
                    <Thead>
                        <Tr>
                            <Th>Entity</Th>
                            <Th>Internal flows</Th>
                        </Tr>
                    </Thead>
                    <TbodyUnified
                        tableState={tableState}
                        colSpan={7}
                        errorProps={{
                            title: 'There was an error loading external ips',
                        }}
                        emptyProps={{
                            message: 'This feature might not be enabled.',
                            children: (
                                <Flex alignSelf={{ default: 'alignSelfCenter' }}>
                                    <ExternalLink>
                                        <a
                                            href={getVersionedDocs(
                                                version,
                                                'operating/visualizing-external-entities'
                                            )}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                        >
                                            Enabling external ip collection
                                        </a>
                                    </ExternalLink>
                                </Flex>
                            ),
                        }}
                        filteredEmptyProps={{
                            title: 'No external ips found',
                            onClearFilters: () => {
                                setSearchFilter({});
                            },
                        }}
                        renderer={({ data }) => (
                            <Tbody>
                                {data.map(({ entity, flowsCount }) => {
                                    return (
                                        <Tr key={entity.id}>
                                            <Td dataLabel="Entity">
                                                <Button
                                                    variant="link"
                                                    isInline
                                                    onClick={() => onExternalIPSelect(entity.id)}
                                                >
                                                    {entity.externalSource.name}
                                                </Button>
                                            </Td>
                                            <Td dataLabel="Internal flows">{flowsCount}</Td>
                                        </Tr>
                                    );
                                })}
                            </Tbody>
                        )}
                    />
                </Table>
            </InnerScrollContainer>
        </>
    );
}

export default ExternalIpsTable;
