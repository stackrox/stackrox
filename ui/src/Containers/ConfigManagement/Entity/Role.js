import React, { useContext } from 'react';
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
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import searchContext from 'Containers/searchContext';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import { SUBJECT_WITH_CLUSTER_FRAGMENT } from 'queries/subject';
import { SERVICE_ACCOUNT_FRAGMENT } from 'queries/serviceAccount';
import getSubListFromEntity from '../List/utilities/getSubListFromEntity';
import EntityList from '../List/EntityList';

const Role = ({ id, entityListType, query }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        where: queryService.objectToWhereClause(query[searchParam])
    };

    const QUERY = gql`
        query k8sRole($id: ID!) {
            clusters {
                id
                k8srole(role: $id) {
                    id
                    name
                    type
                    verbs
                    createdAt
                    roleNamespace {
                        metadata {
                            id
                            name
                        }
                    }
                    serviceAccounts {
                        ${
                            entityListType === entityTypes.SERVICE_ACCOUNT
                                ? '...serviceAccountFields'
                                : 'id'
                        }
                    }
                    subjects {
                        ${
                            entityListType === entityTypes.SUBJECT
                                ? '...subjectWithClusterFields'
                                : 'name'
                        }
                    }
                    rules {
                        apiGroups
                        nonResourceUrls
                        resourceNames
                        resources
                        verbs
                    }
                }
            }
        }

    ${entityListType === entityTypes.SUBJECT ? SUBJECT_WITH_CLUSTER_FRAGMENT : ''}
    ${entityListType === entityTypes.SERVICE_ACCOUNT ? SERVICE_ACCOUNT_FRAGMENT : ''}


    `;
    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                const { clusters } = data;
                if (!clusters || !clusters.length)
                    return <PageNotFound resourceType={entityTypes.ROLE} />;

                const { k8srole: entity } = clusters[0];

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

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            data={getSubListFromEntity(entity, entityListType)}
                            query={query}
                        />
                    );
                }

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
                                        entityId={namespaceId}
                                    />
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Users & Groups"
                                    value={subjects.length}
                                    entityType={entityTypes.SUBJECT}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Service Accounts"
                                    value={serviceAccounts.length}
                                    entityType={entityTypes.SERVICE_ACCOUNT}
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
};

Role.propTypes = entityComponentPropTypes;
Role.defaultProps = entityComponentDefaultProps;

export default Role;
