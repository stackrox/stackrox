import { useCallback } from 'react';
import {
    PageSection,
    Pagination,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import PageTitle from 'Components/PageTitle';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useURLPagination from 'hooks/useURLPagination';
import useRestQuery from 'hooks/useRestQuery';
import useURLSort from 'hooks/useURLSort';
import type { SortOption } from 'types/table';
import { secretTypeLabels } from './secrets.utils';
import { fetchSecretCount, fetchSecrets } from 'services/SecretsService';
import { getTableUIState } from 'utils/getTableUIState';
import { getDateTime } from 'utils/dateUtils';

const sortFields = ['Secret', 'Namespace', 'Cluster', 'Created Time'];

const defaultSortOption: SortOption = {
    field: 'Secret',
    direction: 'asc',
};

function SecretsPage() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(20);
    const { sortOption, getSortParams } = useURLSort({ sortFields, defaultSortOption });

    const fetchSecretsCallback = useCallback(
        () => fetchSecrets({ searchFilter: {}, sortOption, page, perPage }),
        [sortOption, page, perPage]
    );
    const secretsQuery = useRestQuery(fetchSecretsCallback);

    const fetchCountCallback = useCallback(() => fetchSecretCount({}), []);
    const countQuery = useRestQuery(fetchCountCallback);

    const tableState = getTableUIState({
        isLoading: secretsQuery.isLoading,
        data: secretsQuery.data,
        error: secretsQuery.error,
        searchFilter: {},
    });

    return (
        <>
            <PageTitle title="Risk - Secrets" />
            <PageSection>
                <Title headingLevel="h1">Secrets Risk</Title>
            </PageSection>
            <PageSection>
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem variant="pagination" align={{ default: 'alignEnd' }}>
                            <Pagination
                                itemCount={countQuery.data ?? 0}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                            />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
                <Table variant="compact">
                    <Thead>
                        <Tr>
                            <Th modifier="nowrap" sort={getSortParams('Secret')}>
                                Secret
                            </Th>
                            <Th modifier="nowrap">Types</Th>
                            <Th modifier="nowrap" sort={getSortParams('Created Time')}>
                                Created
                            </Th>
                            <Th modifier="nowrap" sort={getSortParams('Cluster')}>
                                Cluster
                            </Th>
                            <Th modifier="nowrap" sort={getSortParams('Namespace')}>
                                Namespace
                            </Th>
                        </Tr>
                    </Thead>
                    <TbodyUnified
                        tableState={tableState}
                        colSpan={5}
                        renderer={({ data }) => (
                            <Tbody>
                                {data.map(
                                    ({ id, name, namespace, clusterName, types, createdAt }) => (
                                        <Tr key={id}>
                                            <Td dataLabel="Secret">{name}</Td>
                                            <Td dataLabel="Types">
                                                {types
                                                    .map((type) => secretTypeLabels[type] ?? type)
                                                    .join(', ')}
                                            </Td>
                                            <Td dataLabel="Created">
                                                {createdAt ? getDateTime(createdAt) : '-'}
                                            </Td>
                                            <Td dataLabel="Cluster">{clusterName}</Td>
                                            <Td dataLabel="Namespace">{namespace}</Td>
                                        </Tr>
                                    )
                                )}
                            </Tbody>
                        )}
                    />
                </Table>
            </PageSection>
        </>
    );
}

export default SecretsPage;
