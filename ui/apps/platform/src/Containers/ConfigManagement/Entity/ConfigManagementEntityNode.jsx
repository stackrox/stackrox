import { useContext } from 'react';
import { gql } from '@apollo/client';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Components/RelatedEntity';
import Metadata from 'Components/Metadata';
import { entityComponentDefaultProps, entityComponentPropTypes } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';
import isGQLLoading from 'utils/gqlLoading';
import queryService from 'utils/queryService';
import { getDateTime } from 'utils/dateUtils';

import EntityList from '../List/EntityList';

const ConfigManagementEntityNode = ({
    id,
    entityListType,
    entityId1,
    query,
    entityContext,
    pagination,
}) => {
    const searchParam = useContext(searchContext);

    const queryObject = { ...query[searchParam] };

    const variables = {
        id,
        query: queryService.getEntityWhereClause(queryObject),
        pagination,
    };

    const QUERY = gql`
        query getNode($id: ID!) {
            node(id: $id) {
                id
                name
                clusterId
                clusterName
                containerRuntimeVersion
                externalIpAddresses
                internalIpAddresses
                joinedAt
                kernelVersion
                kubeletVersion
                osImage
                labels {
                    key
                    value
                }
                annotations {
                    key
                    value
                }
            }
        }
    `;

    return (
        <Query query={QUERY} variables={variables} fetchPolicy="network-only">
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) {
                    return <Loader />;
                }
                if (!data || !data.node) {
                    return <PageNotFound resourceType="NODE" useCase="configmanagement" />;
                }
                const { node } = data;

                const {
                    kernelVersion,
                    kubeletVersion,
                    osImage,
                    labels = [],
                    containerRuntimeVersion,
                    joinedAt,
                    clusterName,
                    clusterId,
                    annotations,
                } = node;

                const metadataKeyValuePairs = [
                    {
                        key: 'Kubelet Version',
                        value: kubeletVersion,
                    },
                    {
                        key: 'Kernel Version',
                        value: kernelVersion,
                    },
                    {
                        key: 'Node OS',
                        value: osImage,
                    },
                    {
                        key: 'Runtime',
                        value: containerRuntimeVersion,
                    },
                    {
                        key: 'Join time',
                        value: joinedAt ? getDateTime(joinedAt) : 'N/A',
                    },
                ];

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={[]}
                            query={query}
                            entityContext={{ ...entityContext, NODE: id }}
                        />
                    );
                }

                return (
                    <div className="w-full">
                        <CollapsibleSection title="Node Summary">
                            <div className="flex mb-4 flex-wrap">
                                <Metadata
                                    className="mx-4 bg-base-100 min-h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                    annotations={annotations}
                                />
                                {!entityContext.CLUSTER && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        name="Cluster"
                                        entityType="CLUSTER"
                                        value={clusterName}
                                        entityId={clusterId}
                                    />
                                )}
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};

ConfigManagementEntityNode.propTypes = entityComponentPropTypes;
ConfigManagementEntityNode.defaultProps = entityComponentDefaultProps;

export default ConfigManagementEntityNode;
