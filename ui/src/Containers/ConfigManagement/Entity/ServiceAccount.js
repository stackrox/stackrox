import React from 'react';
import PropTypes from 'prop-types';
import { SERVICE_ACCOUNT } from 'queries/serviceAccount';
import entityTypes from 'constants/entityTypes';
import nsIcon from 'images/ns-icon.svg';
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

const ServiceAccount = ({ id }) => (
    <Query query={SERVICE_ACCOUNT} variables={{ id }}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            const { serviceAccount } = data;
            if (!serviceAccount) return <PageNotFound resourceType={entityTypes.SERVICE_ACCOUNT} />;
            const metadataKeyValuePairs = [
                { key: 'Automounted', value: serviceAccount.automountToken.toString() },
                {
                    key: 'Created',
                    value: format(serviceAccount.createdAt, 'MM/DD/YYYY H:m:sA')
                }
            ];
            const metadataCounts = [
                { value: serviceAccount.labels.length, text: 'Labels' },
                { value: serviceAccount.secrets.length, text: 'Secrets' },
                { value: serviceAccount.imagePullSecrets.length, text: 'Image Pull Secrets' }
            ];

            return (
                <div className="bg-primary-100 w-full">
                    <CollapsibleSection title="Service Account Details">
                        <div className="flex h-48 mb-4">
                            <Metadata
                                className="w-1/10 flex-grow mx-4 bg-base-100"
                                keyValuePairs={metadataKeyValuePairs}
                                counts={metadataCounts}
                            />
                            <RelatedEntity
                                className="flex-1 mx-4"
                                name="Namespace"
                                icon={nsIcon}
                                value={serviceAccount.namespace}
                            />
                            <RelatedEntityListCount
                                className="flex-1 mx-4"
                                name="Deployments"
                                value={serviceAccount.deployments.length}
                            />
                            <RelatedEntityListCount
                                className="flex-1 mx-4"
                                name="Secrets"
                                value={serviceAccount.secrets.length}
                            />
                            <RelatedEntityListCount
                                className="flex-1 mx-4"
                                name="Roles"
                                value={serviceAccount.roles.length}
                            />
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Service Account Permissions">
                        <div className="flex mb-4">
                            <ClusterScopedPermissions
                                scopedPermissions={serviceAccount.scopedPermissions}
                                className="w-1/3 mx-4 bg-base-100"
                            />
                            <NamespaceScopedPermissions
                                scopedPermissions={serviceAccount.scopedPermissions}
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
    id: PropTypes.string.isRequired
};

export default ServiceAccount;
