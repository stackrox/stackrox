import React, { useState } from 'react';
import {
    Bullseye,
    Button,
    Modal,
    Pagination,
    Spinner,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { gql, useQuery } from '@apollo/client';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useTableSort from 'hooks/patternfly/useTableSort';
import { SearchFilter } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

const deploymentQuery = gql`
    query getDeploymentsForPolicyGeneration($query: String!, $pagination: Pagination!) {
        deployments(query: $query, pagination: $pagination) {
            id
            name
            namespace
        }
    }
`;

const sortFields = ['Deployment', 'Namespace'];
const defaultSortOption = { field: 'Deployment', direction: 'asc' } as const;

export type DeploymentScopeModalProps = {
    searchFilter: SearchFilter;
    scopeDeploymentCount: number;
    isOpen: boolean;
    onClose: () => void;
};

function DeploymentScopeModal({
    searchFilter,
    scopeDeploymentCount,
    isOpen,
    onClose,
}: DeploymentScopeModalProps) {
    const { sortOption, getSortParams } = useTableSort({ sortFields, defaultSortOption });
    const [page, setPage] = useState(1);
    const [perPage, setPerPage] = useState(20);

    const options = {
        skip: !isOpen,
        variables: {
            query: getRequestQueryStringForSearchFilter(searchFilter),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
        },
    };
    const { data, previousData, loading, error } = useQuery<
        {
            deployments: {
                id: string;
                name: string;
                namespace: string;
            }[];
        },
        { query: string }
    >(deploymentQuery, options);

    const deployments = data?.deployments ?? previousData?.deployments ?? [];

    return (
        <Modal
            isOpen={isOpen}
            title="Selected deployment scope"
            variant="small"
            onClose={onClose}
            actions={[
                <Button key="close" onClick={onClose}>
                    Close
                </Button>,
            ]}
        >
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                        <Pagination
                            isCompact
                            itemCount={scopeDeploymentCount}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                if (scopeDeploymentCount < (page - 1) * newPerPage) {
                                    setPage(1);
                                }
                                setPerPage(newPerPage);
                            }}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            {error && (
                <Bullseye>
                    <EmptyStateTemplate
                        title="There was an error loading deployments"
                        headingLevel="h2"
                        icon={ExclamationCircleIcon}
                        iconClassName="pf-u-danger-color-100"
                    >
                        {getAxiosErrorMessage(error.message)}
                    </EmptyStateTemplate>
                </Bullseye>
            )}
            {loading && deployments.length === 0 && (
                <Bullseye>
                    <Spinner aria-label="Loading deployments" />
                </Bullseye>
            )}
            {!error && (
                <TableComposable variant="compact">
                    <Thead noWrap>
                        <Tr>
                            <Th width={50} sort={getSortParams('Deployment')}>
                                Deployment
                            </Th>
                            <Th width={50} sort={getSortParams('Namespace')}>
                                Namespace
                            </Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {deployments.map(({ name, namespace }) => (
                            <Tr key={`${namespace}/${name}`}>
                                <Td dataLabel="Deployment">{name}</Td>
                                <Td dataLabel="Namespace">{namespace}</Td>
                            </Tr>
                        ))}
                    </Tbody>
                </TableComposable>
            )}
        </Modal>
    );
}

export default DeploymentScopeModal;
