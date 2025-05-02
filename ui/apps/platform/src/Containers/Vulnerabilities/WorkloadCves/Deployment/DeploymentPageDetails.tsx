import React from 'react';
import {
    Bullseye,
    Card,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Text,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';

import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import KeyValueListModal from 'Components/KeyValueListModal';
import DateTimeUTCTooltip from 'Components/DateTimeWithUTCTooltip';
import { getDateTime } from 'utils/dateUtils';

export type DeploymentPageDetailsProps = {
    deploymentId: string;
};

type DeploymentDetails = {
    id: string;
    name: string;
    cluster: {
        id: string;
        name: string;
    } | null;
    namespace: string;
    replicas: number;
    created: string | null;
    serviceAccount: string;
    type: string;
    labels: {
        key: string;
        value: string;
    }[];
    annotations: {
        key: string;
        value: string;
    }[];
};

const deploymentDetailsQuery = gql`
    query getDeploymentDetails($id: ID!) {
        deployment(id: $id) {
            id
            name
            cluster {
                id
                name
            }
            namespace
            replicas
            created
            serviceAccount
            type
            labels {
                key
                value
            }
            annotations {
                key
                value
            }
        }
    }
`;

// TODO: We want to potentially create reusable Deployment Details components to be shared between Vuln Management, Violations, and Compliance in the future
// Reference: https://redhat-internal.slack.com/archives/C02MN2N2UG4/p1710184053971889
function DeploymentPageDetails({ deploymentId }: DeploymentPageDetailsProps) {
    const { data, previousData, loading, error } = useQuery<{ deployment: DeploymentDetails }>(
        deploymentDetailsQuery,
        {
            variables: {
                id: deploymentId,
            },
        }
    );

    const deploymentDetailsData = data?.deployment ?? previousData?.deployment;

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>View details about this deployment</Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                component="div"
            >
                {error && (
                    <TableErrorComponent
                        error={error}
                        message="There was an error loading the deployment data"
                    />
                )}
                {loading && !deploymentDetailsData && (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                )}
                {deploymentDetailsData && (
                    <Card className="pf-v5-u-m-md pf-v5-u-p-md" isFlat>
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsLg' }}
                        >
                            <FlexItem>
                                <DescriptionList
                                    isFillColumns
                                    columnModifier={{
                                        md: '3Col',
                                        sm: '1Col',
                                    }}
                                >
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Name</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {deploymentDetailsData.name}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Cluster</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {deploymentDetailsData.cluster?.name || '-'}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Replicas</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {deploymentDetailsData.replicas}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Created</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {deploymentDetailsData.created ? (
                                                <DateTimeUTCTooltip
                                                    datetime={deploymentDetailsData.created}
                                                >
                                                    {getDateTime(deploymentDetailsData.created)}
                                                </DateTimeUTCTooltip>
                                            ) : (
                                                '-'
                                            )}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Namespace</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {deploymentDetailsData.namespace}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Service account</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {deploymentDetailsData.serviceAccount}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Deployment type</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {deploymentDetailsData.type}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Labels</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <KeyValueListModal
                                                type="label"
                                                keyValues={deploymentDetailsData.labels}
                                            />
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Annotations</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <KeyValueListModal
                                                type="annotation"
                                                keyValues={deploymentDetailsData.annotations}
                                            />
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                </DescriptionList>
                            </FlexItem>
                        </Flex>
                    </Card>
                )}
            </PageSection>
        </>
    );
}

export default DeploymentPageDetails;
