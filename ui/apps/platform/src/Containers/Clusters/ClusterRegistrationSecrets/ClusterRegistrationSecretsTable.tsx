import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { ActionsColumn, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { ClusterRegistrationSecret } from 'services/ClustersService';
import { clustersClusterRegistrationSecretsPath } from 'routePaths';

export type ClusterRegistrationSecretsTableProps = {
    hasWriteAccessForClusterRegistrationSecrets: boolean;
    clusterRegistrationSecrets: ClusterRegistrationSecret[];
    setClusterRegistrationSecretToRevoke: (clusterRegistrationSecret: ClusterRegistrationSecret) => void;
};

function ClusterRegistrationSecretsTable({
    hasWriteAccessForClusterRegistrationSecrets,
    clusterRegistrationSecrets,
    setClusterRegistrationSecretToRevoke,
}: ClusterRegistrationSecretsTableProps): ReactElement {
    return (
        <Table variant="compact">
            <Thead>
                <Tr>
                    <Th>Name</Th>
                    <Th>Created by</Th>
                    <Th>Created at</Th>
                    <Th>Expires at</Th>
                    {hasWriteAccessForClusterRegistrationSecrets && (
                        <Th>
                            <span className="pf-v5-screen-reader">Row actions</span>
                        </Th>
                    )}
                </Tr>
            </Thead>
            <Tbody>
                {clusterRegistrationSecrets.map((clusterRegistrationSecret) => {
                    const { createdAt, createdBy, expiresAt, id, name } = clusterRegistrationSecret;

                    return (
                        <Tr key={id}>
                            <Td dataLabel="Name">
                                <Link to={`${clustersClusterRegistrationSecretsPath}/${id}`}>{name}</Link>
                            </Td>
                            <Td dataLabel="Created by">{createdBy.id}</Td>
                            <Td dataLabel="Created at">{createdAt}</Td>
                            <Td dataLabel="Expires at">{expiresAt}</Td>
                            {hasWriteAccessForClusterRegistrationSecrets && (
                                <Td isActionCell>
                                    <ActionsColumn
                                        // menuAppendTo={() => document.body}
                                        items={[
                                            {
                                                title: 'Revoke cluster registration secret',
                                                onClick: () => {
                                                    setClusterRegistrationSecretToRevoke(clusterRegistrationSecret);
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

export default ClusterRegistrationSecretsTable;
