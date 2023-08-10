import React from 'react';
import { Alert, Button, Modal } from '@patternfly/react-core';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import orderBy from 'lodash/orderBy';

import useTableSort from 'hooks/patternfly/useTableSort';
import { EntityScope } from '../utils/simulatorUtils';

export type DeploymentScopeModalProps = {
    entityScope: EntityScope;
    isOpen: boolean;
    onClose: () => void;
};

const sortFields = ['name', 'namespace'];
const defaultSortOption = { field: 'name', direction: 'asc' } as const;

function DeploymentScopeModal({ entityScope, isOpen, onClose }: DeploymentScopeModalProps) {
    const { sortOption, getSortParams } = useTableSort({ sortFields, defaultSortOption });

    const sortedDeployments = orderBy(
        entityScope.deployments,
        sortOption.field,
        sortOption.reversed ? 'desc' : 'asc'
    );

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
            {entityScope.hasAppliedDeploymentFilters && (
                <Alert
                    isInline
                    variant="info"
                    title="The deployment scope for generated policies may be further reduced by filters applied on this page"
                />
            )}
            <TableComposable variant="compact">
                <Thead noWrap>
                    <Tr>
                        <Th width={50} sort={getSortParams('name')}>
                            Deployment
                        </Th>
                        <Th width={50} sort={getSortParams('namespace')}>
                            Namespace
                        </Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {sortedDeployments.map(({ name, namespace }) => (
                        <Tr key={`${namespace}/${name}`}>
                            <Td dataLabel="Deployment">{name}</Td>
                            <Td dataLabel="Namespace">{namespace}</Td>
                        </Tr>
                    ))}
                </Tbody>
            </TableComposable>
        </Modal>
    );
}

export default DeploymentScopeModal;
