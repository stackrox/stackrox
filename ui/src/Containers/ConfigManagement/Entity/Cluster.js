import React from 'react';
import PropTypes from 'prop-types';
import { CLUSTER_QUERY as QUERY } from 'queries/cluster';
import entityTypes from 'constants/entityTypes';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';

const Cluster = ({ id, onRelatedEntityListClick }) => (
    <Query query={QUERY} variables={{ id }}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            const { results: entity } = data;
            if (!entity) return <PageNotFound resourceType={entityTypes.CLUSTER} />;

            const onRelatedEntityListClickHandler = entityListType => () => {
                onRelatedEntityListClick(entityListType);
            };

            const {
                admissionController = false,
                centralApiEndpoint = 'N/A',
                alerts = [],
                nodes = [],
                namespaces = [],
                deployments = [],
                subjects = [],
                serviceAccounts = [],
                k8sroles = []
            } = entity;

            const metadataKeyValuePairs = [
                {
                    key: 'Admission Controller',
                    value: admissionController.toString()
                },
                {
                    key: 'Central API Endpoint',
                    value: centralApiEndpoint
                }
            ];
            const metadataCounts = [{ value: alerts.length, text: 'Alerts' }];

            return (
                <div className="bg-primary-100 w-full">
                    <CollapsibleSection title="Cluster Details">
                        <div className="flex flex-wrap">
                            <Metadata
                                className="flex-grow mx-4 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                counts={metadataCounts}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Nodes"
                                value={nodes.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.NODE)}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Namespaces"
                                value={namespaces.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.NAMESPACE)}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Deployments"
                                value={deployments.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.DEPLOYMENT)}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Users & Groups"
                                value={subjects.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.SUBJECT)}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Service Accounts"
                                value={serviceAccounts.length}
                                onClick={onRelatedEntityListClickHandler(
                                    entityTypes.SERVICE_ACCOUNT
                                )}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Roles"
                                value={k8sroles.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.ROLE)}
                            />
                        </div>
                    </CollapsibleSection>
                </div>
            );
        }}
    </Query>
);

Cluster.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default Cluster;
