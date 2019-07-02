import React from 'react';
import PropTypes from 'prop-types';
import { K8S_ROLE } from 'queries/role';
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
import Rules from 'Containers/ConfigManagement/Entity/widgets/Rules';
import RulePermissions from 'Containers/ConfigManagement/Entity/widgets/RulePermissions';

const Role = ({ id, onRelatedEntityClick, onRelatedEntityListClick }) => (
    <Query query={K8S_ROLE} variables={{ id }}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            const { clusters } = data;
            if (!clusters || !clusters.length)
                return <PageNotFound resourceType={entityTypes.ROLE} />;

            const { k8srole: entity } = clusters[0];

            const onRelatedEntityClickHandler = (entityType, entityId) => () => {
                onRelatedEntityClick(entityType, entityId);
            };

            const onRelatedEntityListClickHandler = entityListType => () => {
                onRelatedEntityListClick(entityListType);
            };

            const {
                type,
                createdAt,
                roleNamespace,
                serviceAccounts = [],
                subjects = [],
                labels = [],
                annotations = [],
                rules
            } = entity;

            const { name: namespaceName, id: namespaceId } = roleNamespace
                ? roleNamespace.metadata
                : {};

            const metadataKeyValuePairs = [
                { key: 'Role Type', value: type },
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
                <div className="bg-primary-100 w-full">
                    <CollapsibleSection title="Role Details">
                        <div className="flex mb-4 flex-wrap">
                            <Metadata
                                className="mx-4 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                counts={metadataCounts}
                            />
                            {roleNamespace && (
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    entityType={entityTypes.NAMESPACE}
                                    name="Namespace Scope"
                                    value={namespaceName}
                                    onClick={onRelatedEntityClickHandler(
                                        entityTypes.NAMESPACE,
                                        namespaceId
                                    )}
                                />
                            )}
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
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Role Permissions And Rules">
                        <div className="flex mb-4">
                            <RulePermissions rules={rules} className="mx-4 bg-base-100" />
                            <Rules rules={rules} className="mx-4 bg-base-100" />
                        </div>
                    </CollapsibleSection>
                </div>
            );
        }}
    </Query>
);

Role.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityClick: PropTypes.func.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default Role;
