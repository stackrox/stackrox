import {
    Bullseye,
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

import DeploymentContainersCard from 'Components/DeploymentContainersCard';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import KeyValueListModal from 'Components/KeyValueListModal';
import type { Container, ContainerConfig } from 'types/deployment.proto';
import type { ImageName } from 'types/image.proto';

import useVulnerabilityState from '../hooks/useVulnerabilityState';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

export type DeploymentPageDetailsProps = {
    deploymentId: string;
};

const emptyImageName: ImageName = {
    registry: '',
    remote: '',
    tag: '',
    fullName: '',
};

type DeploymentContainerDetails = {
    id: string;
    name: string;
    type: Container['type'];
    image: {
        id: string;
        idV2: string;
        notPullable: boolean;
        name: ImageName | null;
    } | null;
    resources: Container['resources'];
    volumes: Container['volumes'];
    secrets: Container['secrets'];
    config: Pick<ContainerConfig, 'args' | 'command'> | null;
    securityContext: Container['securityContext'];
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
    containers: DeploymentContainerDetails[];
};

// DeploymentContainersCard expects a full Container. The details query only loads
// the fields that card renders, so unused config fields stay empty.
function toContainers(containers: DeploymentContainerDetails[] | null | undefined): Container[] {
    return (containers ?? []).map((container) => ({
        id: container.id,
        name: container.name,
        type: container.type,
        image: {
            id: container.image?.id ?? '',
            idV2: container.image?.idV2 ?? '',
            notPullable: container.image?.notPullable ?? false,
            name: container.image?.name ?? emptyImageName,
        },
        resources: container.resources,
        volumes: container.volumes,
        secrets: container.secrets,
        ports: [],
        config: container.config
            ? {
                  args: container.config.args,
                  command: container.config.command,
                  directory: '',
                  user: '',
                  uid: '',
                  appArmorProfile: '',
                  env: [],
              }
            : null,
        securityContext: container.securityContext,
    }));
}

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
            containers {
                id
                name
                type
                image {
                    id
                    idV2
                    notPullable
                    name {
                        registry
                        remote
                        tag
                        fullName
                    }
                }
                resources {
                    cpuCoresRequest
                    cpuCoresLimit
                    memoryMbRequest
                    memoryMbLimit
                }
                volumes {
                    name
                    source
                    destination
                    readOnly
                    type
                    mountPropagation
                }
                secrets {
                    name
                    path
                }
                config {
                    args
                    command
                }
                securityContext {
                    privileged
                    selinux {
                        user
                        role
                        type
                        level
                    }
                    dropCapabilities
                    addCapabilities
                    readOnlyRootFilesystem
                    seccompProfile {
                        type
                        localhostProfile
                    }
                    allowPrivilegeEscalation
                }
            }
        }
    }
`;

// TODO: We want to potentially create reusable Deployment Details components to be shared between Vuln Management, Violations, and Compliance in the future
// Reference: https://redhat-internal.slack.com/archives/C02MN2N2UG4/p1710184053971889
function DeploymentPageDetails({ deploymentId }: DeploymentPageDetailsProps) {
    const { urlBuilder } = useWorkloadCveViewContext();
    const vulnerabilityState = useVulnerabilityState();
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
            <PageSection component="div">
                <Content component="p">View details about this deployment</Content>
            </PageSection>
            <Divider component="div" />
            <PageSection component="div">
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
                        <FlexItem>
                            <DeploymentContainersCard
                                containers={toContainers(deploymentDetailsData.containers)}
                                title="Container configuration"
                                getImageUrl={(imageId) =>
                                    urlBuilder.imageDetails(imageId, vulnerabilityState)
                                }
                            />
                        </FlexItem>
                    </Flex>
                )}
            </PageSection>
        </>
    );
}

export default DeploymentPageDetails;
