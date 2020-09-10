import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import dateFns from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import { fetchDeployment } from 'services/DeploymentsService';
import KeyValuePairs from 'Components/KeyValuePairs';
import CollapsibleCard from 'Components/CollapsibleCard';
import Message from 'Components/Message';
import { portExposureLabels } from 'messages/common';
import SecurityContext from './SecurityContext';
import ContainerConfigurations from './ContainerConfigurations';

const formatDeploymentPorts = (ports) => {
    return ports.map(({ exposure, exposureInfos, ...rest }) => {
        const formattedPort = { ...rest };
        formattedPort.exposure = portExposureLabels[exposure];
        formattedPort.exposureInfos = exposureInfos.map(({ level, ...restInfo }) => {
            return { ...restInfo, level: portExposureLabels[level] };
        });
        return formattedPort;
    });
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
        formatValue: (timestamp) =>
            timestamp ? dateFns.format(timestamp, dateTimeFormat) : 'not available',
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
        formatValue: (v) => v.join(', '),
    },
};

const DeploymentDetails = ({ deployment }) => {
    // attempt to fetch related deployment to selected alert
    const [relatedDeployment, setRelatedDeployment] = useState(deployment);

    useEffect(() => {
        fetchDeployment(deployment.id).then(
            (dep) => setRelatedDeployment(dep),
            () => setRelatedDeployment(null)
        );
    }, [deployment.id, setRelatedDeployment]);

    return (
        <div className="w-full pb-5">
            {!relatedDeployment && (
                <Message
                    type="warn"
                    message="This data is a snapshot of a deployment that no longer exists."
                />
            )}
            <div className="px-3 pt-5">
                <CollapsibleCard title="Overview">
                    <div className="h-full px-3 word-break">
                        <KeyValuePairs data={deployment} keyValueMap={deploymentDetailsMap} />
                    </div>
                </CollapsibleCard>
            </div>
            <ContainerConfigurations deployment={relatedDeployment || deployment} />
            <SecurityContext deployment={deployment} />
        </div>
    );
};

DeploymentDetails.propTypes = {
    deployment: PropTypes.shape({
        id: PropTypes.string.isRequired,
        containers: PropTypes.arrayOf(PropTypes.object),
    }).isRequired,
};

export default DeploymentDetails;
