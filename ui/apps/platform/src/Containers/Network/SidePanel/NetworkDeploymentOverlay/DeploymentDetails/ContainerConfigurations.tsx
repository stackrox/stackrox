import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import lowerCase from 'lodash/lowerCase';
import capitalize from 'lodash/capitalize';

import { vulnManagementPath } from 'routePaths';
import KeyValuePairs from 'Components/KeyValuePairs';

type ContainerConfigMap = {
    command: { label: string };
    args: { label: string };
    ports: { label: string };
    volumes: { label: string };
    secrets: { label: string };
};

const containerConfigMap: ContainerConfigMap = {
    command: { label: 'Commands' },
    args: { label: 'Arguments' },
    ports: { label: 'Ports' },
    volumes: { label: 'Volumes' },
    secrets: { label: 'Secrets' },
};

const getContainerConfigurations = (container): ContainerConfigMap | null => {
    if (!container.config) {
        return null;
    }
    const { command, args, ports, volumes, secrets } = container.config;
    return { command, args, ports, volumes, secrets };
};

const ContainerImage = ({ image }): ReactElement | null => {
    if (!image?.name?.fullName) {
        return null;
    }
    if (image.id === '' || image.notPullable) {
        const unavailableText = image.notPullable
            ? 'image not currently pullable'
            : 'image not available until deployment is running';
        return (
            <div className="flex py-3">
                <div className="font-700 inline pr-1">Image Name:</div>
                <div className="font-600">
                    {image.name.fullName}
                    <span className="italic pl-1">({unavailableText})</span>{' '}
                </div>
            </div>
        );
    }
    // TODO delete type cast when components have types.
    return (
        <div className="py-3 pb-2 leading-normal border-b border-base-300">
            <div className="font-700 inline">Image Name: </div>
            <Link
                className="font-600 text-primary-600 hover:text-primary-800 leading-normal word-break"
                to={`${vulnManagementPath}/image/${image.id as string}`}
            >
                {image.name.fullName}
            </Link>
        </div>
    );
};

const Resources = ({ resources }): ReactElement => {
    if (!resources) {
        return <span className="py-3 font-600 italic">None</span>;
    }
    const resourceMap = {
        cpuCoresRequest: { label: 'CPU Request (cores)' },
        cpuCoresLimit: { label: 'CPU Limit (cores)' },
        memoryMbRequest: { label: 'Memory Request (MB)' },
        memoryMbLimit: { label: 'Memory Limit (MB)' },
    };

    return <KeyValuePairs data={resources} keyValueMap={resourceMap} />;
};

type Volume = Record<string, string>;

const ContainerVolumes = ({ volumes }: { volumes: Volume[] }): ReactElement => {
    if (!volumes?.length) {
        return <span className="py-1 font-600 italic">None</span>;
    }
    return (
        <>
            {volumes.map((volume, idx) => (
                <li
                    key={volume.name}
                    className={`py-2 ${
                        idx === volumes.length - 1 ? '' : 'border-base-300 border-b'
                    }`}
                >
                    {Object.keys(volume).map(
                        (key) =>
                            volume[key] && (
                                <div key={key} className="py-1">
                                    <span className="font-700 pr-1">
                                        {capitalize(lowerCase(key))}:
                                    </span>
                                    <span className="font-600">{volume[key].toString()}</span>
                                </div>
                            )
                    )}
                </li>
            ))}
        </>
    );
};

type Secret = {
    name: string;
    path: string;
};
const ContainerSecrets = ({ secrets }: { secrets: Secret[] }): ReactElement => {
    if (!secrets?.length) {
        return <span className="py-1 font-600">None</span>;
    }
    return (
        <>
            {secrets.map(({ name, path }) => (
                <div key={`${name}-${path}`} className="py-2">
                    <div className="py-1">
                        <span className="font-700 pr-1">Name:</span>
                        <span className="font-600">{name}</span>
                    </div>
                    <div className="py-1">
                        <span className="font-700 pr-1">Container Path:</span>
                        <span className="font-600">{path}</span>
                    </div>
                </div>
            ))}
        </>
    );
};

const ContainerConfigurations = ({ deployment }): ReactElement => {
    let containers: ReactElement;
    if (deployment?.containers?.length) {
        containers = deployment.containers.map((container) => {
            const containerConfigurations = getContainerConfigurations(container);
            const { resources, volumes, secrets } = container;
            return (
                <div key={container.image} data-testid="deployment-container-configuration">
                    <ContainerImage image={container.image} />
                    {containerConfigurations && (
                        <KeyValuePairs
                            data={containerConfigurations}
                            keyValueMap={containerConfigMap}
                        />
                    )}
                    {!!resources && !!volumes && !!secrets && (
                        <>
                            <div className="py-3 border-b border-base-300">
                                <div className="pr-1 font-700 ">Resources:</div>
                                <ul className="ml-2 mt-2 w-full">
                                    <Resources resources={resources} />
                                </ul>
                            </div>
                            <div className="py-3 border-b border-base-300">
                                <div className="pr-1 font-700">Volumes:</div>
                                <ul className="ml-2 mt-2 w-full">
                                    <ContainerVolumes volumes={volumes} />
                                </ul>
                            </div>
                            <div className="py-3 border-b border-base-300">
                                <div className="pr-1 font-700">Secrets:</div>
                                <ul className="ml-2 mt-2 w-full">
                                    <ContainerSecrets secrets={secrets} />
                                </ul>
                            </div>
                        </>
                    )}
                </div>
            );
        });
    } else {
        containers = <span className="py-3 font-600 italic">None</span>;
    }
    return <div className="flex-col h-full px-3">{containers}</div>;
};

export default ContainerConfigurations;
