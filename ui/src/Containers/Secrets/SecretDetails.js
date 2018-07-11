import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import CollapsibleCard from 'Components/CollapsibleCard';
import NoResultsMessage from 'Components/NoResultsMessage';
import KeyValuePairs from 'Components/KeyValuePairs';

const secretDetailsMap = {
    secretId: { label: 'Secret ID' },
    cluster: { label: 'Cluster' },
    namespace: { label: 'Namespace' }
};

const SecretDetails = ({ secret }) => {
    if (!secret) return <NoResultsMessage message="No Secret Details Available" />;
    const secretDetail = {
        secretId: secret.id,
        cluster: secret.clusterRelationship.name,
        namespace: secret.namespaceRelationship.namespace
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
            <div className="px-3 py-4 w-full overflow-y-scroll">
                <div className="bg-white shadow text-primary-600 tracking-wide">
                    <CollapsibleCard title="Deployments">
                        <div className="flex h-full p-3 font-500">
                            {secret.deploymentRelationships.map(deployment => (
                                <div className="flex py-3" key={deployment.id}>
                                    <div className="pr-1 font-600">Deployment Name:</div>
                                    <Link
                                        className="font-500 text-primary-600 hover:text-primary-800"
                                        to={`/main/risk/${deployment.id}`}
                                    >
                                        {deployment.name}
                                    </Link>
                                </div>
                            ))}
                        </div>
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
        clusterRelationship: PropTypes.shape({
            name: PropTypes.string
        }).isRequired,
        namespaceRelationship: PropTypes.shape({
            namespace: PropTypes.string
        }).isRequired,
        deploymentRelationships: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string,
                id: PropTypes.string
            })
        ).isRequired
    }).isRequired
};

export default SecretDetails;
