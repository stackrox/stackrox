import { useEffect, useState } from 'react';
import { Alert } from '@patternfly/react-core';

import { fetchDeployment } from 'services/DeploymentsService';
import CollapsibleCard from 'Components/CollapsibleCard';
import { getDateTime } from 'utils/dateUtils';
import { portExposureLabels } from 'messages/common';
import SecurityContext from './SecurityContext';
import ContainerConfigurations from './ContainerConfigurations';
import KeyValuePairs from './KeyValuePairs';
import type { Deployment, PortConfig } from 'types/deployment.proto';

export function formatDeploymentPorts(ports: Deployment['ports']): Deployment['ports'] {
    return ports.map(({ exposure, exposureInfos, ...rest }) => {
        const formattedPort: PortConfig = { ...rest, exposure: 'UNSET', exposureInfos: [] };
        // @ts-expect-error TODO: The type of `portExposureLabels` is not correct based on declared types.
        formattedPort.exposure = portExposureLabels[exposure] || portExposureLabels.UNSET;
        // @ts-expect-error TODO: The type of `portExposureLabels` is not correct based on declared types.
        formattedPort.exposureInfos = exposureInfos.map(({ level, ...restInfo }) => {
            return { ...restInfo, level: portExposureLabels[level] };
        });
        return formattedPort;
    });
}

const deploymentDetailsMap = {
    id: { label: 'Deployment ID' },
    name: { label: 'Deployment Name' },
    type: { label: 'Deployment Type' },
    clusterName: { label: 'Cluster' },
    namespace: { label: 'Namespace' },
    replicas: { label: 'Replicas' },
    created: {
        label: 'Created',
        formatValue: (timestamp) => (timestamp ? getDateTime(timestamp) : 'not available'),
    },
    labels: { label: 'Labels' },
    annotations: { label: 'Annotations' },
    ports: {
        label: 'Port configuration',
        formatValue: (v: Deployment['ports']) => formatDeploymentPorts(v),
    },
    serviceAccount: { label: 'Service Account' },
    imagePullSecrets: {
        label: 'Image Pull Secrets',
        formatValue: (v: Deployment['imagePullSecrets']) => v.join(', '),
    },
};

type DeploymentDetailsProps = {
    deployment: Deployment;
};

function DeploymentDetails({ deployment }: DeploymentDetailsProps) {
    // attempt to fetch related deployment to selected alert
    const [relatedDeployment, setRelatedDeployment] = useState<Deployment | null>(deployment);

    useEffect(() => {
        fetchDeployment(deployment.id).then(
            (dep) => setRelatedDeployment(dep),
            () => setRelatedDeployment(null)
        );
    }, [deployment.id, setRelatedDeployment]);

    return (
        <div className="w-full pb-5">
            {!relatedDeployment && (
                <Alert
                    variant="warning"
                    isInline
                    title="This data is a snapshot of a deployment that no longer exists"
                    component="p"
                />
            )}
            <div className="px-3 pt-5">
                <CollapsibleCard title="Overview">
                    <div className="h-full px-3 word-break">
                        <KeyValuePairs
                            data={relatedDeployment || deployment}
                            keyValueMap={deploymentDetailsMap}
                        />
                    </div>
                </CollapsibleCard>
            </div>
            <ContainerConfigurations deployment={relatedDeployment || deployment} />
            <SecurityContext deployment={relatedDeployment || deployment} />
        </div>
    );
}

export default DeploymentDetails;
