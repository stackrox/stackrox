import React, { useEffect } from 'react';
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
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';
import { TableUIState } from 'utils/getTableUIState';
import { getVersionedDocs } from 'utils/versioning';
import { ExternalNetworkFlowsMetadata } from 'types/networkFlow.proto';

import IPMatchFilter from '../common/IPMatchFilter';
import { EXTERNAL_SOURCE_ADDRESS_QUERY } from '../NetworkGraph.constants';

export type ExternalIpsTableProps = {
    onExternalIPSelect: (externalIP: string) => void;
    tableState: TableUIState<ExternalNetworkFlowsMetadata>;
    totalEntities: number;
    urlSearchFiltering: UseUrlSearchReturn;
    urlPagination: UseURLPaginationResult;
};

function ExternalIpsTable({
    onExternalIPSelect,
    tableState,
    totalEntities,
    urlSearchFiltering,
    urlPagination,
}: ExternalIpsTableProps) {
    const { version } = useMetadata();
    const { page, perPage, setPage, setPerPage } = urlPagination;
    const { searchFilter, setSearchFilter } = urlSearchFiltering;

    useEffect(() => {
        setPage(1);
    }, [searchFilter, setPage]);

    return (
        <>
            <Toolbar className="pf-v5-u-pb-md pf-v5-u-pt-0">
                <ToolbarContent className="pf-v5-u-px-0">
                    <ToolbarItem className="pf-v5-u-w-100 pf-v5-u-mr-0">
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
                            itemCount={totalEntities}
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
                                setPage(1);
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
