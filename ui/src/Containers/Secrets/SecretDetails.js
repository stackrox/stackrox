import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import CollapsibleCard from 'Components/CollapsibleCard';
import NoResultsMessage from 'Components/NoResultsMessage';
import KeyValuePairs from 'Components/KeyValuePairs';
import * as Icon from 'react-feather';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

export const secretTypeEnumMapping = {
    UNDETERMINED: 'Undetermined',
    PUBLIC_CERTIFICATE: 'Public Certificate',
    CERTIFICATE_REQUEST: 'Certificate Request',
    PRIVACY_ENHANCED_MESSAGE: 'Privacy Enhanced Message',
    OPENSSH_PRIVATE_KEY: 'OpenSSH Private Key',
    PGP_PRIVATE_KEY: 'PGP Private Key',
    EC_PRIVATE_KEY: 'EC Private Key',
    RSA_PRIVATE_KEY: 'RSA Private Key',
    DSA_PRIVATE_KEY: 'DSA Private Key',
    CERT_PRIVATE_KEY: 'Certificate Private Key',
    ENCRYPTED_PRIVATE_KEY: 'Encrypted Private Key'
};

const secretDetailsMap = {
    createdAt: {
        label: 'Created',
        formatValue: timestamp =>
            timestamp ? dateFns.format(timestamp, dateTimeFormat) : 'not available'
    },
    clusterName: { label: 'Cluster' },
    namespace: { label: 'Namespace' },
    type: { label: 'Secret Type' },
    labels: { label: 'Labels' },
    annotations: { label: 'Annotations' }
};

const secretFileCertNameMap = {
    subject: { label: 'Subject' },
    commonName: { label: 'Common Name' },
    country: { label: 'Country' },
    organization: { label: 'Organization' },
    organizationalUnit: { label: 'Organization Unit' },
    locality: { label: 'Locality' },
    province: { label: 'Province' },
    streetAddress: { label: 'Street Address' },
    postalCode: { label: 'Postal Code' },
    names: { label: 'Names' }
};

const secretCertFieldsMap = {
    startDate: {
        label: 'Start Date',
        formatValue: timestamp =>
            timestamp ? dateFns.format(timestamp, dateTimeFormat) : 'not available'
    },
    endDate: {
        label: 'End Date',
        formatValue: timestamp =>
            timestamp ? dateFns.format(timestamp, dateTimeFormat) : 'not available'
    },
    sans: { label: 'SANs' },
    algorithm: { label: 'Algorithm' }
};

const secretFileDetailsMap = {
    type: { label: 'Type', formatValue: d => secretTypeEnumMapping[d] }
};

const getDeploymentRelationships = secret => {
    if (secret.relationship) {
        const relationships = secret.relationship.deploymentRelationships;

        if (relationships && relationships.length !== 0) {
            return relationships.map(deployment => (
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
    }
    return (
        <div className="flex h-full p-3 font-500">
            <span className="py-1 font-500 italic">None</span>
        </div>
    );
};

const renderCert = cert => (
    <div className="w-full h-full font-500">
        <KeyValuePairs data={cert} keyValueMap={secretCertFieldsMap} />
        <span className="font-700 ">Issuer:</span>
        <div className="w-full h-full pl-5 font-500">
            <KeyValuePairs data={cert.issuer} keyValueMap={secretFileCertNameMap} />
        </div>
        <span className="font-700 ">Subject:</span>
        <div className="w-full h-full pl-5 font-500">
            <KeyValuePairs data={cert.subject} keyValueMap={secretFileCertNameMap} />
        </div>
    </div>
);

const renderFileCard = file => (
    <div className="px-3 py-4 w-full overflow-y-scroll">
        <div className="bg-base-100 shadow text-primary-600 tracking-wide">
            <CollapsibleCard title={file.name}>
                <div className="w-full h-full p-3 font-500">
                    <KeyValuePairs data={file} keyValueMap={secretFileDetailsMap} />
                    {file.cert && renderCert(file.cert)}
                </div>
            </CollapsibleCard>
        </div>
    </div>
);

const renderDataDetails = secret => secret.files.map(file => renderFileCard(file));

const SecretDetails = ({ secret }) => {
    if (!secret) return <NoResultsMessage message="No Secret Details Available" />;
    return (
        <div className="h-full w-full bg-base-200">
            <div className="px-3 py-4 w-full overflow-y-scroll">
                <div className="bg-base-100 shadow text-primary-600 tracking-wide">
                    <CollapsibleCard title="Overview">
                        <div className="h-full">
                            <div className="p-3">
                                <KeyValuePairs data={secret} keyValueMap={secretDetailsMap} />
                            </div>
                        </div>
                    </CollapsibleCard>
                </div>
            </div>
            <div data-test-id="deployments-card" className="px-3 py-4 w-full overflow-y-scroll">
                <div className="bg-base-100 shadow text-primary-600 tracking-wide">
                    <CollapsibleCard title="Deployments">
                        {getDeploymentRelationships(secret)}
                    </CollapsibleCard>
                </div>
            </div>
            {secret.files && secret.files.length !== 0 && renderDataDetails(secret)}
        </div>
    );
};

SecretDetails.propTypes = {
    secret: PropTypes.shape({
        name: PropTypes.string.isRequired,
        id: PropTypes.string.isRequired,
        clusterName: PropTypes.string.isRequired,
        namespace: PropTypes.string.isRequired,
        relationship: PropTypes.shape({
            deploymentRelationships: PropTypes.arrayOf(
                PropTypes.shape({
                    name: PropTypes.string,
                    id: PropTypes.string
                })
            )
        })
    }).isRequired
};

export default SecretDetails;
