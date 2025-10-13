import React from 'react';
import {
    Bullseye,
    Card,
    CardBody,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    PageSection,
    Spinner,
    Text,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import { getDateTime } from 'utils/dateUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import ExpandableLabelSection from '../../components/ExpandableLabelSection';
import useNodeExtendedDetails from './useNodeExtendedDetails';

export type NodePageDetailsProps = {
    nodeId: string;
};

function NodePageDetails({ nodeId }: NodePageDetailsProps) {
    const { data, loading, error } = useNodeExtendedDetails(nodeId);

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>View details about this node</Text>
            </PageSection>
            <PageSection isFilled className="pf-v5-u-display-flex pf-v5-u-flex-direction-column">
                <Card>
                    <CardBody>
                        {error ? (
                            <Bullseye>
                                <EmptyStateTemplate
                                    title="There was an error loading the node details"
                                    headingLevel="h2"
                                    icon={ExclamationCircleIcon}
                                    iconClassName="pf-v5-u-danger-color-100"
                                >
                                    {getAxiosErrorMessage(error)}
                                </EmptyStateTemplate>
                            </Bullseye>
                        ) : loading ? (
                            <Bullseye>
                                <Spinner size="xl" />
                            </Bullseye>
                        ) : (
                            data && (
                                <Flex
                                    direction={{ default: 'column' }}
                                    spaceItems={{ default: 'spaceItemsXl' }}
                                >
                                    <DescriptionList
                                        columnModifier={{ default: '1Col', lg: '2Col' }}
                                    >
                                        <DescriptionListGroup>
                                            <DescriptionListTerm>Cluster</DescriptionListTerm>
                                            <DescriptionListDescription>
                                                {data.node.cluster.name}
                                            </DescriptionListDescription>
                                        </DescriptionListGroup>
                                        {data.node.containerRuntimeVersion && (
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>
                                                    Container runtime
                                                </DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {data.node.containerRuntimeVersion}
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                        )}
                                        {data.node.joinedAt && (
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>Join time</DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {getDateTime(data.node.joinedAt)}
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                        )}
                                        {data.node.scanTime && (
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>Scan time</DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {getDateTime(data.node.scanTime)}
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                        )}
                                        {data.node.kernelVersion && (
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>
                                                    Kernel version
                                                </DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {data.node.kernelVersion}
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                        )}
                                        {data.node.kubeletVersion && (
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>Kubelet</DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {data.node.kubeletVersion}
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                        )}
                                    </DescriptionList>
                                    <ExpandableLabelSection
                                        toggleText="Labels"
                                        labels={data.node.labels}
                                    />
                                    <ExpandableLabelSection
                                        toggleText="Annotations"
                                        labels={data.node.annotations}
                                    />
                                </Flex>
                            )
                        )}
                    </CardBody>
                </Card>
            </PageSection>
        </>
    );
}

export default NodePageDetails;
