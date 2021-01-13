import React, { ReactElement, useEffect, useState } from 'react';

import CollapsibleCard from 'Components/CollapsibleCard';
import KeyValuePairs from 'Components/KeyValuePairs';
import { portExposureLabels } from 'messages/common';
import { fetchDeployment } from 'services/DeploymentsService';
import { getDate } from 'utils/dateUtils';
import ContainerConfigurations from './ContainerConfigurations';
import SecurityContext from './SecurityContext';

export const formatDeploymentPorts = (ports): string[] => {
    return ports.map(({ exposure, exposureInfos, ...rest }) => {
        const formattedPort = { ...rest };
        formattedPort.exposure = portExposureLabels[exposure] || portExposureLabels.UNSET;
        formattedPort.exposureInfos = exposureInfos.map(({ level, ...restInfo }) => {
            return { ...restInfo, level: portExposureLabels[level] };
        });
        return formattedPort;
    }) as string[];
};

const deploymentDetailsMap = {
    id: { label: 'Deployment ID' },
    name: { label: 'Deployment Name' },
    type: { label: 'Deployment Type' },
    clusterName: { label: 'Cluster' },
    namespace: { label: 'Namespace' },
    replicas: { label: 'Replicas' },
    created: {
        label: 'Created',
        formatValue: (timestamp): string => (timestamp ? getDate(timestamp) : 'Not available'),
    },
    labels: { label: 'Labels' },
    annotations: { label: 'Annotations' },
    ports: {
        label: 'Port configuration',
        formatValue: (v) => formatDeploymentPorts(v),
    },
    serviceAccount: { label: 'Service Account' },
    imagePullSecrets: {
        label: 'Image Pull Secrets',
        formatValue: (v) => v.join(', ') as string,
    },
};

export type DeploymentDetailsProps = {
    deploymentId: string;
};

function DeploymentDetails({ deploymentId }: DeploymentDetailsProps): ReactElement {
    const [deploymentDetails, setDeplopymentDetails] = useState({});

    useEffect(() => {
        fetchDeployment(deploymentId).then(
            (deployment) => setDeplopymentDetails(deployment),
            () => setDeplopymentDetails({})
        );
    }, [deploymentId, setDeplopymentDetails]);
    return (
        <div className="flex flex-col bg-base-100 rounded border border-base-400 overflow-y-scroll p-3 w-full h-full">
            <CollapsibleCard title="Overview" cardClassName="border border-base-400 mb-3">
                <div className="h-full px-3 word-break">
                    <KeyValuePairs data={deploymentDetails} keyValueMap={deploymentDetailsMap} />
                </div>
            </CollapsibleCard>
            <CollapsibleCard
                title="Container configuration"
                cardClassName="border border-base-400 mb-3"
            >
                <ContainerConfigurations deployment={deploymentDetails} />
            </CollapsibleCard>
            <CollapsibleCard title="Security context" cardClassName="border border-base-400">
                <SecurityContext deployment={deploymentDetails} />
            </CollapsibleCard>
        </div>
    );
}

export default DeploymentDetails;
