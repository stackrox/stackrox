import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import CollapsibleCard from 'Components/CollapsibleCard';
import NoResultsMessage from 'Components/NoResultsMessage';
import KeyValuePairs from 'Components/KeyValuePairs';
import * as Icon from 'react-feather';

const secretDetailsMap = {
    secretId: { label: 'Secret ID' },
    cluster: { label: 'Cluster' },
    namespace: { label: 'Namespace' }
};

const getDeploymentRelationships = secret => {
    if (secret.deploymentRelationships && secret.deploymentRelationships.length !== 0) {
        return secret.deploymentRelationships.map(deployment => (
            <div key={deployment.id} className="w-full h-full p-3 font-500">
                <Icon.Circle className="h-2 w-2 mr-3" />
                <Link
                    className="font-500 text-primary-600 hover:text-primary-800"
                    to={`/main/risk/${deployment.id}`}
                >
                    {deployment.name}
                </Link>
            </div>
        ));
    }
    return (
        <div className="flex h-full p-3 font-500">
            <span className="py-1 font-500 italic">None</span>
        </div>
    );
};

const SecretDetails = ({ secret }) => {
    if (!secret) return <NoResultsMessage message="No Secret Details Available" />;
    const secretDetail = {
        secretId: secret.id,
        cluster: secret.cluster,
        namespace: secret.namespace
    };

    return (
        <div className="h-full w-full bg-base-100">
            <div className="px-3 py-4 w-full overflow-y-scroll">
                <div className="bg-white shadow text-primary-600 tracking-wide">
                    <CollapsibleCard title="Overview">
                        <div className="h-full">
                            <div className="p-3">
                                <KeyValuePairs data={secretDetail} keyValueMap={secretDetailsMap} />
                            </div>
                        </div>
                    </CollapsibleCard>
                </div>
            </div>
            <div data-test-id="deployments-card" className="px-3 py-4 w-full overflow-y-scroll">
                <div className="bg-white shadow text-primary-600 tracking-wide">
                    <CollapsibleCard title="Deployments">
                        {getDeploymentRelationships(secret)}
                    </CollapsibleCard>
                </div>
            </div>
        </div>
    );
};

SecretDetails.propTypes = {
    secret: PropTypes.shape({
        name: PropTypes.string.isRequired,
        id: PropTypes.string.isRequired,
        cluster: PropTypes.string.isRequired,
        namespace: PropTypes.string.isRequired,
        deploymentRelationships: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string,
                id: PropTypes.string
            })
        ).isRequired
    }).isRequired
};

export default SecretDetails;
