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
    Label,
    LabelGroup,
    PageSection,
    Spinner,
    Text,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';

import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
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
            <PageSection component="div" variant="light" className="pf-u-py-md pf-u-px-xl">
                <Text>View details about this deployment</Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
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
                    <Card className="pf-u-m-md pf-u-p-md" isFlat>
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
                                            {deploymentDetailsData.created
                                                ? getDateTime(deploymentDetailsData.created)
                                                : '-'}
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
                                </DescriptionList>
                            </FlexItem>
                            <Divider component="div" />
                            <FlexItem>
                                <DescriptionList
                                    isFillColumns
                                    columnModifier={{
                                        md: '2Col',
                                        sm: '1Col',
                                    }}
                                >
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Labels</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <LabelGroup>
                                                {deploymentDetailsData.labels.map((label) => {
                                                    return (
                                                        <Label>
                                                            {label.key}: {label.value}
                                                        </Label>
                                                    );
                                                })}
                                            </LabelGroup>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Annotations</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <LabelGroup>
                                                {deploymentDetailsData.annotations.map(
                                                    (annotation) => {
                                                        return (
                                                            <Label>
                                                                {annotation.key}: {annotation.value}
                                                            </Label>
                                                        );
                                                    }
                                                )}
                                            </LabelGroup>
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
