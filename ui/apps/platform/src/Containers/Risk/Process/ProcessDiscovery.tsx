import { useEffect, useState } from 'react';
import { Alert, Bullseye, Spinner } from '@patternfly/react-core';
import uniqBy from 'lodash/uniqBy';

import usePermissions from 'hooks/usePermissions';
import { fetchProcessesInBaseline } from 'services/ProcessBaselineService';
import { fetchProcesses } from 'services/ProcessService';
import type { ProcessNameAndContainerNameGroup } from 'services/ProcessService';
import type { ProcessBaseline } from 'types/processBaseline.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import EventTimelineOverview from '../EventTimeline/EventTimelineOverview';
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
        <div>
            {hasReadAccessForAlert && (
                <>
                    <h3 className="border-b border-base-500 pb-2 mx-3 my-5">Event Timeline</h3>
                    <div className="px-3">
                        <EventTimelineOverview deploymentId={deploymentId} />
                    </div>
                </>
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
            <h3 className="border-b border-base-500 pb-2 mx-3 my-5">
                History of Running Processes
            </h3>
            <ProcessDiscoveryCards
                deploymentId={deploymentId}
                processGroups={processGroups}
                processEpoch={processEpoch}
                setProcessEpoch={setProcessEpoch}
            />
            {errorMessageForBaselines && (
                <div className="pt-2 mx-3">
                    <Alert
                        variant="warning"
                        isInline
                        title="Unable to fetch process baselines"
                        component="p"
                    >
                        <p>{errorMessageForBaselines}</p>
                    </Alert>
                </div>
            )}
            {Array.isArray(processBaselines) && (
                <>
                    <h3 className="border-b border-base-500 pb-2 mx-3 my-5">
                        Spec Container Baselines
                    </h3>
                    <SpecificationBaselineList
                        processBaselines={processBaselines}
                        processEpoch={processEpoch}
                        setProcessEpoch={setProcessEpoch}
                    />
                </>
            )}
        </div>
    );
}

export default ProcessDiscovery;
