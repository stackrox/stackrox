import React from 'react';
import { Link } from 'react-router-dom';
import lowerCase from 'lodash/lowerCase';
import capitalize from 'lodash/capitalize';

import { vulnManagementPath } from 'routePaths';
import CollapsibleCard from 'Components/CollapsibleCard';

import KeyValuePairs from './KeyValuePairs';

const containerConfigMap = {
    command: { label: 'Commands' },
    args: { label: 'Arguments' },
    ports: { label: 'Ports' },
    volumes: { label: 'Volumes' },
    secrets: { label: 'Secrets' },
};

const getContainerConfigurations = (container) => {
    if (!container.config) {
        return null;
    }
    const { command, args, ports, volumes, secrets } = container.config;
    return { command, args, ports, volumes, secrets };
};

const ContainerImage = ({ image }) => {
    if (!image?.name?.fullName) {
        return null;
    }
    if (image.id === '' || image.notPullable) {
        const unavailableText = image.notPullable
            ? 'image not currently pullable'
            : 'image not available until deployment is running';
        return (
            <div className="flex py-3">
                <div className="pr-1 ">Image Name:</div>
                <div className="font-500">
                    {image.name.fullName}
                    <span className="pl-1">({unavailableText})</span>{' '}
                </div>
            </div>
        );
    }
    return (
        <div className="py-3 pb-2 leading-normal border-b border-base-300">
            <div className="font-700 inline">Image Name: </div>
            <Link
                className="hover:text-primary-800 leading-normal word-break"
                to={`${vulnManagementPath}/image/${image.id}`}
            >
                {image.name.fullName}
            </Link>
        </div>
    );
};

const Resources = ({ resources }) => {
    if (!resources) {
        return <span className="py-3">None</span>;
    }
    const resourceMap = {
        cpuCoresRequest: { label: 'CPU Request (cores)' },
        cpuCoresLimit: { label: 'CPU Limit (cores)' },
        memoryMbRequest: { label: 'Memory Request (MB)' },
        memoryMbLimit: { label: 'Memory Limit (MB)' },
    };

    return <KeyValuePairs data={resources} keyValueMap={resourceMap} />;
};

const ContainerVolumes = ({ volumes }) => {
    if (!volumes?.length) {
        return <span className="py-1">None</span>;
    }
    return volumes.map((volume, idx) => (
        <li
            key={volume.name}
            className={`py-2 ${idx === volumes.length - 1 ? '' : 'border-base-300 border-b'}`}
        >
            {Object.keys(volume).map(
                (key) =>
                    volume[key] && (
                        <div key={key} className="py-1">
                            <span className="font-700 pr-1">{capitalize(lowerCase(key))}:</span>
                            <span>{volume[key].toString()}</span>
                        </div>
                    )
            )}
        </li>
    ));
};

const ContainerSecrets = ({ secrets }) => {
    if (!secrets?.length) {
        return <span className="py-1">None</span>;
    }
    return secrets.map(({ name, path }) => (
        <div key={name} className="py-2">
            <div className="py-1">
                <span className="font-700 pr-1">Name:</span>
                <span>{name}</span>
            </div>
            <div className="py-1">
                <span className="font-700 pr-1">Container Path:</span>
                <span>{path}</span>
            </div>
        </div>
    ));
};

const ContainerConfigurations = ({ deployment }) => {
    const title = 'Container configuration';
    let containers = [];
    if (deployment.containers) {
        containers = deployment.containers.map((container) => {
            const data = getContainerConfigurations(container);
            const { id, resources, volumes, secrets } = container;
            return (
                <div key={id} data-testid="deployment-container-configuration">
                    <ContainerImage image={container.image} />
                    {data && <KeyValuePairs data={data} keyValueMap={containerConfigMap} />}
                    {!!resources && !!volumes && !!secrets && (
                        <>
                            <div className="py-3 border-b border-base-300">
                                <div className="pr-1 font-700 ">Resources:</div>
                                <div className="ml-2 mt-2 w-full">
                                    <Resources resources={resources} />
                                </div>
                            </div>
                            <div className="py-3 border-b border-base-300">
                                <div className="pr-1 font-700">Volumes:</div>
                                <ul className="ml-2 mt-2 w-full">
                                    <ContainerVolumes volumes={volumes} />
                                </ul>
                            </div>
                            <div className="py-3 border-b border-base-300">
                                <div className="pr-1 font-700">Secrets:</div>
                                <div className="ml-2 mt-2 w-full">
                                    <ContainerSecrets secrets={secrets} />
                                </div>
                            </div>
                        </>
                    )}
                </div>
            );
        });
    } else {
        containers = <span className="py-1">None</span>;
    }
    return (
        <div className="px-3 pt-5">
            <CollapsibleCard title={title}>
                <div className="h-full px-3">{containers}</div>
            </CollapsibleCard>
        </div>
    );
};

export default ContainerConfigurations;
