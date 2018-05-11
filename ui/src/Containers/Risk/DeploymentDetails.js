import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import lowerCase from 'lodash/lowerCase';
import capitalize from 'lodash/capitalize';

import KeyValuePairs from 'Components/KeyValuePairs';
import CollapsibleCard from 'Components/CollapsibleCard';

const deploymentDetailsMap = {
    id: { label: 'Deployment ID' },
    clusterName: { label: 'Cluster' },
    namespace: { label: 'Namespace' },
    replicas: { label: 'Replicas' },
    labels: { label: 'Labels' },
    ports: { label: 'Port configuration' },
    volume: { label: 'Volume' }
};

const containerConfigMap = {
    args: { label: 'Args' },
    command: { label: 'Command' },
    directory: { label: 'Directory' },
    env: { label: 'Environment' },
    uid: { label: 'User ID' },
    user: { label: 'User' }
};

class DeploymentDetails extends Component {
    static propTypes = {
        deployment: PropTypes.shape({ id: PropTypes.string.isRequired }).isRequired
    };

    renderOverview() {
        const title = 'Overview';
        return (
            <div className="px-3 py-4">
                <div className="bg-white shadow text-primary-600 tracking-wide border border-base-200">
                    <CollapsibleCard title={title}>
                        <div className="h-full p-3">
                            <KeyValuePairs
                                data={this.props.deployment}
                                keyValueMap={deploymentDetailsMap}
                            />
                        </div>
                    </CollapsibleCard>
                </div>
            </div>
        );
    }

    renderContainerImage = image => {
        if (!image || !image.name || !image.name.fullName) return null;
        return (
            <div className="flex py-3">
                <div className="pr-1">Image Name:</div>
                <Link
                    className="font-500 text-primary-600 hover:text-primary-800"
                    to={`/main/images/${image.name.sha}`}
                >
                    {image.name.fullName}
                </Link>
            </div>
        );
    };

    renderContainerVolumes = volumes => {
        if (!volumes || !volumes.length) return null;
        return volumes.map((volume, idx) => (
            <li
                key={idx}
                className={`py-2 ${idx === volumes.length - 1 ? '' : 'border-base-300 border-b'}`}
            >
                {Object.keys(volume).map(
                    (key, i) =>
                        volume[key] && (
                            <div key={`${volume.name}-${i}`} className="py-1 font-500">
                                <span className=" pr-1">{capitalize(lowerCase(key))}:</span>
                                <span className="text-accent-400 italic">
                                    {volume[key].toString()}
                                </span>
                            </div>
                        )
                )}
            </li>
        ));
    };

    renderContainerConfigurations = () => {
        const { deployment } = this.props;
        const title = 'Container configuration';
        return (
            <div className="px-3 py-4">
                <div className="bg-white shadow text-primary-600 tracking-wide border border-base-200">
                    <CollapsibleCard title={title}>
                        <div className="h-full p-3">
                            {deployment.containers.map((container, index) => {
                                if (!container.config) return null;
                                return (
                                    <div key={index}>
                                        <KeyValuePairs
                                            data={container.config}
                                            keyValueMap={containerConfigMap}
                                        />
                                        <div className="flex py-3">
                                            <div className="pr-1">Mounts:</div>
                                            <ul className="-ml-8 mt-4 w-full list-reset">
                                                {this.renderContainerVolumes(container.volumes)}
                                            </ul>
                                        </div>
                                        {this.renderContainerImage(container.image)}
                                    </div>
                                );
                            })}
                        </div>
                    </CollapsibleCard>
                </div>
            </div>
        );
    };

    render() {
        return (
            <div>
                {this.renderOverview()}
                {this.renderContainerConfigurations()}
            </div>
        );
    }
}

export default DeploymentDetails;
