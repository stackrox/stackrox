import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { ActionsColumn, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import DateDistance from 'Components/DateDistance';
import type { ClusterRegistrationSecret, InitBundleAttribute } from 'services/ClustersService';
import { clustersClusterRegistrationSecretsPath } from 'routePaths';

const createdByDisplayKeys = ['name', 'username', 'email'];

function getCreatedByDisplayValue(createdBy: ClusterRegistrationSecret['createdBy']): string {
    const match = createdByDisplayKeys
        .map((key) => createdBy.attributes.find((attr) => attr.key === key))
        .find((attr): attr is InitBundleAttribute => attr !== undefined);

    return match?.value ?? createdBy.id;
}

function getRegistrationsDisplay(
    maxRegistrations: string,
    registrationsInitiated: string[]
): string {
    const max = parseInt(maxRegistrations, 10);
    if (!maxRegistrations || max === 0) {
        return 'Unlimited';
    }
    return `${registrationsInitiated.length} / ${max} registered`;
}

export type ClusterRegistrationSecretsTableProps = {
    hasWriteAccessForClusterRegistrationSecrets: boolean;
    clusterRegistrationSecrets: ClusterRegistrationSecret[];
    setClusterRegistrationSecretToRevoke: (
        clusterRegistrationSecret: ClusterRegistrationSecret
    ) => void;
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
                    <Th>Expires in</Th>
                    <Th>Registrations</Th>
                    {hasWriteAccessForClusterRegistrationSecrets && (
                        <Th screenReaderText="Row actions" />
                    )}
                </Tr>
            </Thead>
            <Tbody>
                {clusterRegistrationSecrets.map((clusterRegistrationSecret) => {
                    const {
                        createdBy,
                        expiresAt,
                        id,
                        maxRegistrations,
                        name,
                        registrationsInitiated,
                    } = clusterRegistrationSecret;

                    const createdByDisplay = getCreatedByDisplayValue(createdBy);
                    const isExpired = new Date(expiresAt) < new Date();

                    return (
                        <Tr key={id}>
                            <Td dataLabel="Name">
                                <Link to={`${clustersClusterRegistrationSecretsPath}/${id}`}>
                                    {name}
                                </Link>
                            </Td>
                            <Td dataLabel="Created by">{createdByDisplay}</Td>
                            <Td dataLabel="Expires in">
                                {isExpired ? (
                                    'Expired'
                                ) : (
                                    <DateDistance date={expiresAt} asPhrase={false} />
                                )}
                            </Td>
                            <Td dataLabel="Registrations">
                                {getRegistrationsDisplay(maxRegistrations, registrationsInitiated)}
                            </Td>
                            {hasWriteAccessForClusterRegistrationSecrets && (
                                <Td isActionCell>
                                    <ActionsColumn
                                        items={[
                                            {
                                                title: 'Revoke cluster registration secret',
                                                onClick: () => {
                                                    setClusterRegistrationSecretToRevoke(
                                                        clusterRegistrationSecret
                                                    );
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
