import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { ActionsColumn, TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { ClusterInitBundle } from 'services/ClustersService';
import { clustersInitBundlesPath } from 'routePaths';

export type InitBundlesTableProps = {
    hasWriteAccessForInitBundles: boolean;
    initBundles: ClusterInitBundle[];
    setInitBundleToRevoke: (initBundle: ClusterInitBundle) => void;
};

function InitBundlesTable({
    hasWriteAccessForInitBundles,
    initBundles,
    setInitBundleToRevoke,
}: InitBundlesTableProps): ReactElement {
    return (
        <TableComposable variant="compact">
            <Thead>
                <Tr>
                    <Th>Name</Th>
                    <Th>Created by</Th>
                    <Th>Created at</Th>
                    <Th>Expires at</Th>
                    {hasWriteAccessForInitBundles && <Td />}
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
                            {hasWriteAccessForInitBundles && (
                                <Td>
                                    <ActionsColumn
                                        menuAppendTo={() => document.body}
                                        items={[
                                            {
                                                title: 'Revoke bundle',
                                                onClick: () => {
                                                    setInitBundleToRevoke(initBundle);
                                                },
                                            },
                                        ]}
                                    />
                                </Td>
                            )}
                        </Tr>
                    );
                })}
            </Tbody>
        </TableComposable>
    );
}

export default InitBundlesTable;
