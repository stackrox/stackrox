import React from 'react';
import PropTypes from 'prop-types';
import { CLUSTER_QUERY as QUERY } from 'queries/cluster';
import entityTypes from 'constants/entityTypes';
import { distanceInWordsToNow } from 'date-fns';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import NodesWithFailedControls from './widgets/NodesWithFailedControls';
import DeploymentsWithFailedPolicies from './widgets/DeploymentsWithFailedPolicies';

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
                alertsCount = 'N/A',
                nodes = [],
                namespaces = [],
                deployments = [],
                subjects = [],
                serviceAccounts = [],
                k8sroles = [],
                status: { orchestratorMetadata = null }
            } = entity;

            const { buildDate, version = 'N/A' } = orchestratorMetadata;

            const metadataKeyValuePairs = [
                {
                    key: 'K8s version',
                    value: version
                },
                {
                    key: 'Cluster age',
                    value: buildDate ? distanceInWordsToNow(buildDate) : 'N/A'
                }
            ];
            const metadataCounts = [{ value: alertsCount, text: 'Alerts' }];

            return (
                <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                    <CollapsibleSection title="Cluster Details">
                        <div className="flex flex-wrap pdf-page">
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
                    <CollapsibleSection title="Cluster Findings">
                        <div className="flex pdf-page pdf-stretch rounded relative rounded mb-4 ml-4 mr-4">
                            <Tabs
                                hasTabSpacing
                                headers={[{ text: 'Policies' }, { text: 'CIS Controls' }]}
                            >
                                <TabContent>
                                    <DeploymentsWithFailedPolicies />
                                </TabContent>
                                <TabContent>
                                    <NodesWithFailedControls />
                                </TabContent>
                            </Tabs>
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
