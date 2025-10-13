import React from 'react';
import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { ActionsColumn, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { ClusterInitBundle } from 'services/ClustersService';
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
        <Table variant="compact">
            <Thead>
                <Tr>
                    <Th>Name</Th>
                    <Th>Created by</Th>
                    <Th>Created at</Th>
                    <Th>Expires at</Th>
                    {hasWriteAccessForInitBundles && (
                        <Th>
                            <span className="pf-v5-screen-reader">Row actions</span>
                        </Th>
                    )}
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
                                <Td isActionCell>
                                    <ActionsColumn
                                        // menuAppendTo={() => document.body}
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
        </Table>
    );
}

export default InitBundlesTable;
