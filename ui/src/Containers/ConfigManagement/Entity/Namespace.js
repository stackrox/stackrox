import React from 'react';
import PropTypes from 'prop-types';
import { NAMESPACE_QUERY as QUERY } from 'queries/namespace';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';
import queryService from 'modules/queryService';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import DeploymentsWithFailedPolicies from 'Containers/ConfigManagement/Entity/widgets/DeploymentsWithFailedPolicies';

const Namespace = ({ id, onRelatedEntityListClick }) => (
    <Query query={QUERY} variables={{ id }}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            const { results: entity } = data;
            if (!entity) return <PageNotFound resourceType={entityTypes.NAMESPACE} />;

            const onRelatedEntityListClickHandler = entityListType => () => {
                onRelatedEntityListClick(entityListType);
            };

            const {
                metadata = {},
                numDeployments = 0,
                numSecrets = 0,
                policyCount = 0,
                imageCount = 0
            } = entity;

            const { name, creationTime, labels = [] } = metadata;

            const metadataKeyValuePairs = [
                {
                    key: 'Created',
                    value: creationTime ? format(creationTime, dateTimeFormat) : 'N/A'
                }
            ];
            const metadataCounts = [{ value: labels.length, text: 'Labels' }];

            return (
                <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                    <CollapsibleSection title="Namespace Details">
                        <div className="flex flex-wrap pdf-page">
                            <Metadata
                                className="mx-4 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                counts={metadataCounts}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Deployments"
                                value={numDeployments}
                                onClick={onRelatedEntityListClickHandler(entityTypes.DEPLOYMENT)}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Secrets"
                                value={numSecrets}
                                onClick={onRelatedEntityListClickHandler(entityTypes.SECRET)}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Policies"
                                value={policyCount}
                                onClick={onRelatedEntityListClickHandler(entityTypes.POLICY)}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Images"
                                value={imageCount}
                                onClick={onRelatedEntityListClickHandler(entityTypes.IMAGE)}
                            />
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Namespace Findings">
                        <div className="flex pdf-page pdf-stretch rounded relative rounded mb-4 ml-4 mr-4">
                            <DeploymentsWithFailedPolicies
                                query={queryService.objectToWhereClause({ Namespace: name })}
                            />
                        </div>
                    </CollapsibleSection>
                </div>
            );
        }}
    </Query>
);

Namespace.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default Namespace;
