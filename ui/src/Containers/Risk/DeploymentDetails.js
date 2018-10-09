import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import lowerCase from 'lodash/lowerCase';
import capitalize from 'lodash/capitalize';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import KeyValuePairs from 'Components/KeyValuePairs';
import CollapsibleCard from 'Components/CollapsibleCard';

const deploymentDetailsMap = {
    id: { label: 'Deployment ID' },
    type: { label: 'Deployment Type' },
    clusterName: { label: 'Cluster' },
    namespace: { label: 'Namespace' },
    replicas: { label: 'Replicas' },
    updatedAt: {
        label: 'Updated',
        formatValue: timestamp =>
            timestamp ? dateFns.format(timestamp, dateTimeFormat) : 'not available'
    },
    labels: { label: 'Labels' },
    annotations: { label: 'Annotations' },
    ports: { label: 'Port configuration' },
    serviceAccount: { label: 'Service Account' },
    imagePullSecrets: {
        label: 'Image Pull Secrets',
        formatValue: v => v.join(', ')
    }
};

const containerConfigMap = {
    command: { label: 'Commands' },
    args: { label: 'Arguments' },
    ports: { label: 'Ports' },
    volumes: { label: 'Volumes' },
    secrets: { label: 'Secrets' }
};

const containerSecurityContextMap = {
    privileged: { label: 'Privileged' },
    add_capabilities: { label: 'Add Capabilities' },
    drop_capabilities: { label: 'Drop Capabilities' }
};

class DeploymentDetails extends Component {
    static propTypes = {
        deployment: PropTypes.shape({ id: PropTypes.string.isRequired }).isRequired
    };

    getContainerConfigurations = container => {
        if (!container.config) return null;
        const { command, args, ports, volumes, secrets } = container.config;
        return { command, args, ports, volumes, secrets };
    };

    getSecurityContext = container => {
        if (!container.securityContext) return null;
        const { privileged, add_capabilities, drop_capabilities } = container.securityContext; // eslint-disable-line
        return { privileged, add_capabilities, drop_capabilities };
    };

    renderOverview() {
        const title = 'Overview';
        return (
            <div className="px-3 pt-5">
                <CollapsibleCard title={title}>
                    <div className="h-full px-3 word-break">
                        <KeyValuePairs
                            data={this.props.deployment}
                            keyValueMap={deploymentDetailsMap}
                        />
                    </div>
                </CollapsibleCard>
            </div>
        );
    }

    renderContainerImage = image => {
        if (!image || !image.name || !image.name.fullName) return null;
        if (image.id === '') {
            return (
                <div className="flex py-3">
                    <div className="pr-1 ">Image Name:</div>
                    <div className="font-500">
                        {image.name.fullName}
                        <span className="italic pl-1">
                            (image not available until deployment is running)
                        </span>{' '}
                    </div>
                </div>
            );
        }
        return (
            <div className="py-3 pb-2 leading-normal border-b border-base-300">
                <div className="font-700 inline">Image Name: </div>
                <Link
                    className="font-600 text-primary-600 hover:text-primary-800 leading-normal word-break"
                    to={`/main/images/${image.id}`}
                >
                    {image.name.fullName}
                </Link>
            </div>
        );
    };

    renderResources = resources => {
        if (!resources) return <span className="py-3 font-600 italic">None</span>;
        const resourceMap = {
            cpuCoresRequest: { label: 'CPU Request (cores)' },
            cpuCoresLimit: { label: 'CPU Limit (cores)' },
            memoryMbRequest: { label: 'Memory Request (MB)' },
            memoryMbLimit: { label: 'Memory Limit (MB)' }
        };

        return <KeyValuePairs data={resources} keyValueMap={resourceMap} />;
    };

    renderContainerVolumes = volumes => {
        if (!volumes || !volumes.length) return <span className="py-1 font-600 italic">None</span>;
        return volumes.map((volume, idx) => (
            <li
                key={idx}
                className={`py-2 ${idx === volumes.length - 1 ? '' : 'border-base-300 border-b'}`}
            >
                {Object.keys(volume).map(
                    (key, i) =>
                        volume[key] && (
                            <div key={`${volume.name}-${i}`} className="py-1 font-600">
                                <span className=" pr-1">{capitalize(lowerCase(key))}:</span>
                                <span className="text-accent-800 italic">
                                    {volume[key].toString()}
                                </span>
                            </div>
                        )
                )}
            </li>
        ));
    };

    renderContainerSecrets = secrets => {
        if (!secrets || !secrets.length) return <span className="py-1 font-600 italic">None</span>;
        return secrets.map((secret, idx) => (
            <div key={idx} className="py-2">
                <div key={`${secret.name}-${idx}`} className="py-1 font-600">
                    <span className=" pr-1">Name:</span>
                    <span className="text-accent-800 italic">{secret.name}</span>
                </div>
                <div key={`${secret.path}-${idx}`} className="py-1 font-600">
                    <span className=" pr-1">Container Path:</span>
                    <span className="text-accent-800 italic">{secret.path}</span>
                </div>
            </div>
        ));
    };

    renderContainerConfigurations = () => {
        const { deployment } = this.props;
        const title = 'Container configuration';
        let containers = [];
        if (deployment.containers) {
            containers = deployment.containers.map((container, index) => {
                const data = this.getContainerConfigurations(container);
                return (
                    <div key={index} data-test-id="deployment-container-configuration">
                        {this.renderContainerImage(container.image)}
                        {data && <KeyValuePairs data={data} keyValueMap={containerConfigMap} />}
                        <div className="py-3 border-b border-base-300">
                            <div className="pr-1 font-700 ">Resources:</div>
                            <ul className="ml-2 mt-2 w-full list-reset">
                                {this.renderResources(container.resources)}
                            </ul>
                        </div>
                        <div className="py-3 border-b border-base-300">
                            <div className="pr-1 font-700">Mounts:</div>
                            <ul className="ml-2 mt-2 w-full list-reset">
                                {this.renderContainerVolumes(container.volumes)}
                            </ul>
                        </div>
                        <div className="py-3 border-b border-base-300">
                            <div className="pr-1 font-700">Secrets:</div>
                            <ul className="ml-2 mt-2 w-full list-reset">
                                {this.renderContainerSecrets(container.secrets)}
                            </ul>
                        </div>
                    </div>
                );
            });
        } else {
            containers = <span className="py-1 font-600 italic">None</span>;
        }
        return (
            <div className="px-3 pt-5">
                <CollapsibleCard title={title}>
                    <div className="h-full px-3">{containers}</div>
                </CollapsibleCard>
            </div>
        );
    };

    renderSecurityContext = () => {
        const { deployment } = this.props;
        const title = 'Security Context';
        let containers = [];
        if (deployment.containers) {
            containers = deployment.containers
                .filter(container => container.securityContext)
                .map((container, index) => {
                    const data = this.getSecurityContext(container);
                    if (data === {}) return null;
                    return (
                        <div key={index}>
                            {data && (
                                <KeyValuePairs
                                    data={data}
                                    keyValueMap={containerSecurityContextMap}
                                />
                            )}
                        </div>
                    );
                });
            if (!containers.length) containers = <span className="py-3 font-600 italic">None</span>;
        } else {
            containers = <span className="py-3 font-600 italic">None</span>;
        }
        return (
            <div className="px-3 pt-5">
                <div className="bg-base-100 text-primary-600 tracking-wide">
                    <CollapsibleCard title={title}>
                        <div className="flex h-full px-3">{containers}</div>
                    </CollapsibleCard>
                </div>
            </div>
        );
    };

    render() {
        return (
            <div className="w-full pb-5">
                {this.renderOverview()}
                {this.renderContainerConfigurations()}
                {this.renderSecurityContext()}
            </div>
        );
    }
}

export default DeploymentDetails;
