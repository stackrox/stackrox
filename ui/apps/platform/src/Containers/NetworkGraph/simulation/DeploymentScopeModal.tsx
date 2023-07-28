import React from 'react';
import { Button, Modal } from '@patternfly/react-core';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import orderBy from 'lodash/orderBy';

import useTableSort from 'hooks/patternfly/useTableSort';

export type DeploymentScopeModalProps = {
    deployments: {
        namespace: string;
        name: string;
    }[];
    isOpen: boolean;
    onClose: () => void;
};

const sortFields = ['name', 'namespace'];
const defaultSortOption = { field: 'name', direction: 'asc' } as const;

function DeploymentScopeModal({ deployments, isOpen, onClose }: DeploymentScopeModalProps) {
    const { sortOption, getSortParams } = useTableSort({ sortFields, defaultSortOption });

    const sortedDeployments = orderBy(
        deployments,
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
