import { useState } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Alert, Bullseye, ExpandableSection, Pagination, Spinner } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql, useQuery } from '@apollo/client';

import DateDistance from 'Components/DateDistance';
import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import type { Pagination as PaginationParam } from 'services/types';
import { getPaginationParams } from 'utils/searchUtils';

const RELATED_DEPLOYMENTS_QUERY = gql`
    query getImageDeployments($id: ID!, $pagination: Pagination) {
        imageV2(id: $id) {
            id
            deploymentCount
            deployments(pagination: $pagination) {
                id
                name
                clusterName
                namespace
                created
            }
        }
    }
`;

type RelatedDeploymentsData = {
    imageV2: {
        id: string;
        deploymentCount: number;
        deployments: {
            id: string;
            name: string;
            clusterName: string;
            namespace: string;
            created: string | null;
        }[];
    } | null;
};

type RelatedDeploymentsVars = {
    id: string;
    pagination: PaginationParam;
};

const DEFAULT_PER_PAGE = 10;

export type RelatedDeploymentsProps = {
    imageId: string;
};

function RelatedDeployments({ imageId }: RelatedDeploymentsProps) {
    const [page, setPage] = useState(1);
    const [perPage, setPerPage] = useState(DEFAULT_PER_PAGE);

    const { data, previousData, loading, error } = useQuery<
        RelatedDeploymentsData,
        RelatedDeploymentsVars
    >(RELATED_DEPLOYMENTS_QUERY, {
        variables: {
            id: imageId,
            pagination: getPaginationParams({ page, perPage }),
        },
    });

    const imageData = data?.imageV2 ?? previousData?.imageV2;
    const deploymentCount = imageData?.deploymentCount ?? 0;
    const deployments = imageData?.deployments ?? [];

    if (error) {
        return (
            <Alert
                variant="warning"
                title="Unable to load related deployments"
                component="p"
                isInline
            >
                {error.message}
            </Alert>
        );
    }

    return (
        <ExpandableSection toggleText={`Related deployments (${deploymentCount})`}>
            {loading && !imageData ? (
                <Bullseye>
                    <Spinner size="md" />
                </Bullseye>
            ) : deploymentCount === 0 ? (
                <Alert
                    variant="info"
                    title="No deployments currently use this image"
                    component="p"
                    isInline
                >
                    Deployments that previously used this image may have been deleted or updated.
                </Alert>
            ) : (
                <>
                    <Pagination
                        itemCount={deploymentCount}
                        page={page}
                        perPage={perPage}
                        onSetPage={(_, newPage) => setPage(newPage)}
                        onPerPageSelect={(_, newPerPage) => {
                            setPerPage(newPerPage);
                            setPage(1);
                        }}
                        isCompact
                    />
                    <Table variant="compact">
                        <Thead noWrap>
                            <Tr>
                                <Th>Deployment</Th>
                                <Th>Cluster</Th>
                                <Th>Namespace</Th>
                                <Th>Created</Th>
                            </Tr>
                        </Thead>
                        <Tbody>
                            {deployments.map(({ id, name, clusterName, namespace, created }) => (
                                <Tr key={id}>
                                    <Td dataLabel="Deployment">
                                        <Link
                                            to={`${vulnerabilitiesWorkloadCvesPath}/deployments/${id}`}
                                        >
                                            {name}
                                        </Link>
                                    </Td>
                                    <Td dataLabel="Cluster">{clusterName}</Td>
                                    <Td dataLabel="Namespace">{namespace}</Td>
                                    <Td dataLabel="Created">
                                        <DateDistance date={created} />
                                    </Td>
                                </Tr>
                            ))}
                        </Tbody>
                    </Table>
                </>
            )}
        </ExpandableSection>
    );
}

export default RelatedDeployments;
