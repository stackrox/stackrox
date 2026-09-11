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
import { getDateTime } from 'utils/dateUtils';

import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import KeyValueListModal from 'Components/KeyValueListModal';
import useDeploymentDetails from '../hooks/useDeploymentDetails';

export type DeploymentPageDetailsProps = {
    deploymentId: string;
};

// TODO: We want to potentially create reusable Deployment Details components to be shared between Vuln Management, Violations, and Compliance in the future
// Reference: https://redhat-internal.slack.com/archives/C02MN2N2UG4/p1710184053971889
function DeploymentPageDetails({ deploymentId }: DeploymentPageDetailsProps) {
    const { data, loading, error } = useDeploymentDetails(deploymentId);

    const deploymentDetailsData = data?.deployment;

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
                    </Flex>
                )}
            </PageSection>
        </>
    );
}

export default DeploymentPageDetails;
