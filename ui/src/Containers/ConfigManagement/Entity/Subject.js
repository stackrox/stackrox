import React from 'react';
import PropTypes from 'prop-types';
import { SUBJECT_QUERY } from 'queries/subject';
import entityTypes from 'constants/entityTypes';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import ClusterScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/ClusterScopedPermissions';
import NamespaceScopedPermissions from 'Containers/ConfigManagement/Entity/widgets/NamespaceScopedPermissions';

const processSubjectDataByClusters = data => {
    const entity = data.clusters.reduce(
        (acc, curr) => {
            const { subject, type, clusterAdmin, ...rest } = curr.subject;
            return {
                subject,
                type,
                clusterAdmin,
                clusters: [...acc.clusters, { ...rest }]
            };
        },
        { clusters: [] }
    );
    return entity;
};

const Subject = ({ id, onRelatedEntityListClick }) => (
    <Query query={SUBJECT_QUERY} variables={{ id }}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            if (!data.clusters || !data.clusters.length)
                return <PageNotFound resourceType={entityTypes.SUBJECT} />;

            const entity = processSubjectDataByClusters(data);

            const onRelatedEntityListClickHandler = entityListType => () => {
                onRelatedEntityListClick(entityListType);
            };

            const { type, clusterAdmin, clusters = [] } = entity;

            const rolesAcrossAllClusters = clusters.reduce((acc, { roles }) => {
                return [...acc, roles];
            }, []);

            const scopedPermissionsAcrossAllClusters = clusters.reduce(
                (acc, { scopedPermissions }) => {
                    return [...acc, ...scopedPermissions];
                },
                []
            );

            const metadataKeyValuePairs = [
                { key: 'Role type', value: type },
                {
                    key: 'Cluster Admin Role',
                    value: clusterAdmin ? 'Enabled' : 'Disabled'
                }
            ];
            const metadataCounts = [];

            return (
                <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                    <CollapsibleSection title="Subject Details">
                        <div className="flex mb-4 flex-wrap pdf-page">
                            <Metadata
                                className="mx-4 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                counts={metadataCounts}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Roles"
                                value={rolesAcrossAllClusters.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.ROLE)}
                            />
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Subject Permissions">
                        <div className="flex mb-4 pdf-page pdf-stretch">
                            <ClusterScopedPermissions
                                scopedPermissions={scopedPermissionsAcrossAllClusters}
                                className="mx-4 bg-base-100"
                            />
                            <NamespaceScopedPermissions
                                scopedPermissions={scopedPermissionsAcrossAllClusters}
                                className="flex-grow mx-4 bg-base-100"
                            />
                        </div>
                    </CollapsibleSection>
                </div>
            );
        }}
    </Query>
);

Subject.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default Subject;
