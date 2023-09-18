import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { ClusterInitBundle } from 'services/ClustersService';
import { clustersInitBundlesPath } from 'routePaths';

export type InitBundlesTableProps = {
    initBundles: ClusterInitBundle[];
};

function InitBundlesTable({ initBundles }: InitBundlesTableProps): ReactElement {
    return (
        <TableComposable variant="compact">
            <Thead>
                <Tr>
                    <Th>Name</Th>
                    <Th>Created by</Th>
                    <Th>Created at</Th>
                    <Th>Expires at</Th>
                </Tr>
            </Thead>
            <Tbody>
                {initBundles.map((initBundle) => {
                    const { createdAt, createdBy, expiresAt, id, name } = initBundle;

                    return (
                        <Tr key={id}>
                            <Td dataLabel="Name">
                                <Link to={`${clustersInitBundlesPath}/${id}`}>{name}</Link>
                            </Td>
                            <Td dataLabel="Created by">{createdBy.id}</Td>
                            <Td dataLabel="Created at">{createdAt}</Td>
                            <Td dataLabel="Expires at">{expiresAt}</Td>
                        </Tr>
                    );
                })}
            </Tbody>
        </TableComposable>
    );
}

export default InitBundlesTable;
