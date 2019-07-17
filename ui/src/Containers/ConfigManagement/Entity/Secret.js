import React from 'react';
import PropTypes from 'prop-types';
import { SECRET as QUERY } from 'queries/secret';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import CollapsibleRow from 'Components/CollapsibleRow';
import Widget from 'Components/Widget';

const SecretDataMetadata = ({ metadata }) => {
    if (!metadata) return null;
    const { startDate, endDate, issuer = {}, sans, subject = {} } = metadata;
    const {
        commonName: issuerCommonName = 'N/A',
        names: issuerNames,
        organizationUnit = 'N/A'
    } = issuer;
    const { commonName: subjectCommonName = 'N/A', names: subjectNames } = subject;
    return (
        <div className="flex flex-row">
            <Widget
                header="Timeframe"
                className="m-4"
                bodyClassName="flex flex-col p-4 leading-normal"
            >
                <div>
                    <span className="font-700 mr-4">Start Date:</span>
                    <span>{startDate ? format(startDate, dateTimeFormat) : 'N/A'}</span>
                </div>
                <div>
                    <span className="font-700 mr-4">End Date:</span>
                    <span>{endDate ? format(endDate, dateTimeFormat) : 'N/A'}</span>
                </div>
            </Widget>
            <Widget
                header="Issuer"
                className="m-4"
                bodyClassName="flex flex-col p-4 leading-normal"
            >
                <div>
                    <span className="font-700 mr-4">Common Name:</span>
                    <span>{issuerCommonName}</span>
                </div>
                <div>
                    <span className="font-700 mr-4">Name(s):</span>
                    <span>{issuerNames ? issuerNames.join(', ') : 'None'}</span>
                </div>
                <div>
                    <span className="font-700 mr-4">Organization Unit:</span>
                    <span>{organizationUnit}</span>
                </div>
            </Widget>
            <Widget
                header="Subject"
                className="m-4"
                bodyClassName="flex flex-col p-4 leading-normal"
            >
                <div>
                    <span className="font-700 mr-4">Common Name:</span>
                    <span>{subjectCommonName}</span>
                </div>
                <div>
                    <span className="font-700 mr-4">Name(s):</span>
                    <span>{subjectNames ? subjectNames.join(', ') : 'None'}</span>
                </div>
            </Widget>
            <Widget header="Sans" className="m-4" bodyClassName="flex flex-col p-4 leading-normal">
                <div>
                    <span className="font-700 mr-4">Sans:</span>
                    <span>{sans ? sans.join(', ') : 'None'}</span>
                </div>
            </Widget>
        </div>
    );
};

SecretDataMetadata.propTypes = {
    metadata: PropTypes.shape()
};

SecretDataMetadata.defaultProps = {
    metadata: null
};

const SecretValues = ({ files, deployments }) => {
    const filesWithoutImagePullSecrets = files.filter(
        // eslint-disable-next-line
        file => file.metadata && file.metadata.__typename !== 'ImagePullSecret'
    );
    const widgetHeader = `${filesWithoutImagePullSecrets.length} files across ${
        deployments.length
    } deployment(s)`;
    const secretValues = filesWithoutImagePullSecrets.map((file, i) => {
        const { name, type, metadata } = file;
        const { algorithm } = metadata || {};
        const collapsibleRowHeader = (
            <div className="flex flex-1 w-full">
                <div className="flex flex-1">{name}</div>
                {type && (
                    <div className="border-l border-base-400 px-2 capitalize">
                        {type.replace(/_/g, ' ').toLowerCase()}
                    </div>
                )}
                {algorithm && <div className="border-l border-base-400 px-2">{algorithm}</div>}
            </div>
        );
        return (
            <CollapsibleRow key={i} header={collapsibleRowHeader} isCollapsible={!!metadata}>
                <SecretDataMetadata metadata={metadata} />
            </CollapsibleRow>
        );
    });
    return (
        <Widget header={widgetHeader} bodyClassName="flex flex-col">
            {secretValues}
        </Widget>
    );
};

SecretValues.propTypes = {
    files: PropTypes.arrayOf(PropTypes.shape).isRequired,
    deployments: PropTypes.arrayOf(PropTypes.shape).isRequired
};

const Secret = ({ id, onRelatedEntityClick, onRelatedEntityListClick }) => (
    <Query query={QUERY} variables={{ id }}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            const { secret: entity } = data;
            if (!entity) return <PageNotFound resourceType={entityTypes.SECRET} />;

            const onRelatedEntityClickHandler = (entityType, entityId) => () => {
                onRelatedEntityClick(entityType, entityId);
            };

            const onRelatedEntityListClickHandler = entityListType => () => {
                onRelatedEntityListClick(entityListType);
            };

            const {
                createdAt,
                labels = [],
                annotations = [],
                deployments = [],
                clusterName,
                clusterId,
                files
            } = entity;

            const metadataKeyValuePairs = [
                {
                    key: 'Created',
                    value: createdAt ? format(createdAt, dateTimeFormat) : 'N/A'
                }
            ];
            const metadataCounts = [
                { value: labels.length, text: 'Labels' },
                { value: annotations.length, text: 'Annotations' }
            ];

            return (
                <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                    <CollapsibleSection title="Secret Details">
                        <div className="flex mb-4 flex-wrap pdf-page">
                            <Metadata
                                className="mx-4 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                counts={metadataCounts}
                            />
                            <RelatedEntity
                                className="mx-4 min-w-48 h-48 mb-4"
                                entityType={entityTypes.CLUSTER}
                                name="Cluster"
                                value={clusterName}
                                onClick={onRelatedEntityClickHandler(
                                    entityTypes.CLUSTER,
                                    clusterId
                                )}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Deployments"
                                value={deployments.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.DEPLOYMENT)}
                            />
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Secret Values">
                        <div className="flex pdf-page pdf-stretch mb-4 ml-4 mr-4">
                            <SecretValues files={files} deployments={deployments} />
                        </div>
                    </CollapsibleSection>
                </div>
            );
        }}
    </Query>
);

Secret.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityClick: PropTypes.func.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default Secret;
