import { useEffect, useState } from 'react';
import { Alert, Bullseye, Flex, FlexItem, Spinner, Title } from '@patternfly/react-core';
import uniqBy from 'lodash/uniqBy';

import usePermissions from 'hooks/usePermissions';
import { fetchProcessesInBaseline } from 'services/ProcessBaselineService';
import { fetchProcesses } from 'services/ProcessService';
import type { ProcessNameAndContainerNameGroup } from 'services/ProcessService';
import type { ProcessBaseline } from 'types/processBaseline.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import EventTimelineOverview from './EventTimelineOverview';
import SpecificationBaselineList from './SpecificationBaselineList';
import ProcessDiscoveryCards from './ProcessDiscoveryCards';
import { getClusterIdAndNamespaceFromGroupedProcesses } from './process.utils';

function getBaselineRequests(
    clusterId: string,
    deploymentId: string,
    namespace: string,
    uniqueContainerNames: string[]
) {
    return uniqueContainerNames.map((containerName) => {
        const queryStr = `key.clusterId=${clusterId}&key.namespace=${namespace}&key.deploymentId=${deploymentId}&key.containerName=${containerName}`;
        return fetchProcessesInBaseline(queryStr);
    });
}

export type ProcessDiscoveryProps = {
    deploymentId: string;
};

function ProcessDiscovery({ deploymentId }: ProcessDiscoveryProps) {
    const [processEpoch, setProcessEpoch] = useState(0);
    const [processGroups, setProcessGroups] = useState<ProcessNameAndContainerNameGroup[]>([]);
    const [processBaselines, setProcessBaselines] = useState<ProcessBaseline[] | null>(null);
    const [errorMessage, setErrorMessage] = useState('');
    const [errorMessageForBaselines, setErrorMessageForBaselines] = useState('');
    const [isFetching, setIsFetching] = useState(false);
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForAlert = hasReadAccess('Alert');

    useEffect(() => {
        if (processEpoch === 0) {
            setIsFetching(true); // render spinner only for initial request
        }
        fetchProcesses(deploymentId)
            .then((groups) => {
                setProcessGroups(groups);

                if (Array.isArray(groups)) {
                    const { clusterId, namespace } =
                        getClusterIdAndNamespaceFromGroupedProcesses(groups);
                    const uniqueContainerNames = uniqBy(groups, 'containerName').map(
                        (x) => x.containerName
                    );
                    if (clusterId && namespace && uniqueContainerNames.length) {
                        const requests = getBaselineRequests(
                            clusterId,
                            deploymentId,
                            namespace,
                            uniqueContainerNames
                        );
                        Promise.all(requests)
                            .then(setProcessBaselines)
                            .catch((error) => {
                                setErrorMessageForBaselines(getAxiosErrorMessage(error));
                            });
                    }
                }
            })
            .catch((error) => {
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsFetching(false);
            });
    }, [deploymentId, setProcessGroups, processEpoch]);

    if (isFetching) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
            {hasReadAccessForAlert && (
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <FlexItem>
                        <Title headingLevel="h2">Event timeline</Title>
                    </FlexItem>
                    <FlexItem>
                        <EventTimelineOverview deploymentId={deploymentId} />
                    </FlexItem>
                </Flex>
            )}
            {errorMessage && (
                <Alert variant="warning" isInline title="No processes discovered" component="p">
                    <p>{errorMessage}</p>
                    <p>
                        The selected deployment may not have running pods, or Collector may not be
                        running in your cluster.
                    </p>
                    <p>It is recommended to check the logs for more information.</p>
                </Alert>
            )}
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                <FlexItem>
                    <Title headingLevel="h2">History of running processes</Title>
                </FlexItem>
                <FlexItem>
                    <ProcessDiscoveryCards
                        deploymentId={deploymentId}
                        processGroups={processGroups}
                        processEpoch={processEpoch}
                        setProcessEpoch={setProcessEpoch}
                    />
                </FlexItem>
            </Flex>
            {errorMessageForBaselines && (
                <Alert
                    variant="warning"
                    isInline
                    title="Unable to fetch process baselines"
                    component="p"
                >
                    <p>{errorMessageForBaselines}</p>
                </Alert>
            )}
            {Array.isArray(processBaselines) && (
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <FlexItem>
                        <Title headingLevel="h2">Spec container baselines</Title>
                    </FlexItem>
                    <FlexItem>
                        <SpecificationBaselineList
                            processBaselines={processBaselines}
                            processEpoch={processEpoch}
                            setProcessEpoch={setProcessEpoch}
                        />
                    </FlexItem>
                </Flex>
            )}
        </Flex>
    );
}

export default ProcessDiscovery;
