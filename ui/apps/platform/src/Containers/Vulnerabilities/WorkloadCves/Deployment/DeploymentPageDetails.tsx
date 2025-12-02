import {
    Bullseye,
    Card,
    Content,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';

import { getDateTime } from 'utils/dateUtils';

import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import KeyValueListModal from 'Components/KeyValueListModal';

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
            <PageSection
                hasBodyWrapper={false}
                component="div"
                className="pf-v6-u-py-md pf-v6-u-px-xl"
            >
                <Content component="p">View details about this deployment</Content>
            </PageSection>
            <Divider component="div" />
            <PageSection
                hasBodyWrapper={false}
                className="pf-v6-u-display-flex pf-v6-u-flex-direction-column pf-v6-u-flex-grow-1"
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
                    <Card className="pf-v6-u-m-md pf-v6-u-p-md">
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
