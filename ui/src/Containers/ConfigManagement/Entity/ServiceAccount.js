import React from 'react';
import PropTypes from 'prop-types';
import { SERVICE_ACCOUNT } from 'queries/serviceAccount';
import entityTypes from 'constants/entityTypes';
import nsIcon from 'images/ns-icon.svg';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import ClusterScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/ClusterScopedPermissions';
import NamespaceScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/NamespaceScopedPermissions';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';

const ServiceAccount = ({ id, onRelatedEntityClick, onRelatedEntityListClick }) => (
    <Query query={SERVICE_ACCOUNT} variables={{ id }}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            const { serviceAccount: entity } = data;
            if (!entity) return <PageNotFound resourceType={entityTypes.SERVICE_ACCOUNT} />;

            const onRelatedEntityClickHandler = (entityType, entityId) => () => {
                onRelatedEntityClick(entityType, entityId);
            };

            const onRelatedEntityListClickHandler = entityListType => () => {
                onRelatedEntityListClick(entityListType);
            };

            const {
                automountToken = false,
                createdAt,
                labels = [],
                secrets = [],
                imagePullSecrets = [],
                deployments = [],
                roles = [],
                saNamespace: { metadata = {} },
                scopedPermissions = []
            } = entity;

            const { name: namespaceName, id: namespaceId } = metadata;

            const metadataKeyValuePairs = [
                { key: 'Automounted', value: automountToken.toString() },
                {
                    key: 'Created',
                    value: createdAt ? format(createdAt, dateTimeFormat) : 'N/A'
                }
            ];
            const metadataCounts = [
                { value: labels.length, text: 'Labels' },
                { value: secrets.length, text: 'Secrets' },
                { value: imagePullSecrets.length, text: 'Image Pull Secrets' }
            ];

            return (
                <div className="bg-primary-100 w-full">
                    <CollapsibleSection title="Service Account Details">
                        <div className="flex mb-4 flex-wrap">
                            <Metadata
                                className="mx-4 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                counts={metadataCounts}
                            />
                            <RelatedEntity
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Namespace"
                                icon={nsIcon}
                                value={namespaceName}
                                onClick={onRelatedEntityClickHandler(
                                    entityTypes.NAMESPACE,
                                    namespaceId
                                )}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Deployments"
                                value={deployments.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.DEPLOYMENT)}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Secrets"
                                value={secrets.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.SECRET)}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Roles"
                                value={roles.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.ROLE)}
                            />
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Service Account Permissions">
                        <div className="flex mb-4">
                            <ClusterScopedPermissions
                                scopedPermissions={scopedPermissions}
                                className="w-1/3 mx-4 bg-base-100"
                            />
                            <NamespaceScopedPermissions
                                scopedPermissions={scopedPermissions}
                                className="w-2/3 flex-grow mx-4 bg-base-100"
                            />
                        </div>
                    </CollapsibleSection>
                </div>
            );
        }}
    </Query>
);

ServiceAccount.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityClick: PropTypes.func.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default ServiceAccount;
