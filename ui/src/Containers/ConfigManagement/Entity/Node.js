import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { format } from 'date-fns';
import { entityToColumns } from 'constants/listColumns';

import NoResultsMessage from 'Components/NoResultsMessage';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import TableWidget from 'Containers/ConfigManagement/Entity/widgets/TableWidget';
import searchContext from 'Containers/searchContext';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import { standardLabels } from 'messages/standards';
import { CONTROL_FRAGMENT } from 'queries/controls';
import getControlsWithStatus from '../List/utilities/getControlsWithStatus';
import EntityList from '../List/EntityList';

const Node = ({ id, entityListType, entityId1, query, entityContext }) => {
    const searchParam = useContext(searchContext);

    const queryObject = { ...query[searchParam] };
    if (!queryObject.Standard) queryObject.Standard = 'CIS';

    const variables = {
        id,
        query: queryService.getEntityWhereClause(queryObject)
    };

    const QUERY = gql`
        query getNode($id: ID!, $query: String) {
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
                osImage
                labels {
                    key
                    value
                }
                annotations {
                    key
                    value
                }
                complianceResults(query: $query) {
                    ...controlFields
                }
            }
        }
        ${CONTROL_FRAGMENT}
    `;

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader transparent />;
                if (!data || !data.node) return <PageNotFound resourceType={entityTypes.NODE} />;
                const { node } = data;

                const {
                    kernelVersion,
                    osImage,
                    labels = [],
                    containerRuntimeVersion,
                    joinedAt,
                    clusterName,
                    clusterId,
                    annotations,
                    complianceResults = []
                } = node;

                const metadataKeyValuePairs = [
                    {
                        key: 'K8s Version',
                        value: kernelVersion
                    },
                    {
                        key: 'Node OS',
                        value: osImage
                    },
                    {
                        key: 'Runtime',
                        value: containerRuntimeVersion
                    },
                    {
                        key: 'Join time',
                        value: joinedAt ? format(joinedAt, dateTimeFormat) : 'N/A'
                    }
                ];

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={getControlsWithStatus(complianceResults)}
                            query={query}
                            entityContext={{ ...entityContext, [entityTypes.NODE]: id }}
                        />
                    );
                }

                const failedComplianceResults = complianceResults
                    .filter(cr => cr.value.overallState === 'COMPLIANCE_STATE_FAILURE')
                    .map(cr => ({
                        ...cr,
                        control: {
                            ...cr.control,
                            standard: standardLabels[cr.control.standardId]
                        }
                    }));

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Node Details">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                    annotations={annotations}
                                />
                                {!entityContext.CLUSTER && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 h-48 mb-4"
                                        name="Cluster"
                                        entityType={entityTypes.CLUSTER}
                                        value={clusterName}
                                        entityId={clusterId}
                                    />
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="CIS Controls"
                                    value={complianceResults.length}
                                    entityType={entityTypes.CONTROL}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Node Findings">
                            <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                                {failedComplianceResults.length === 0 && (
                                    <NoResultsMessage
                                        message="No nodes failing controls in this cluster"
                                        className="p-6 shadow"
                                        icon="info"
                                    />
                                )}
                                {failedComplianceResults.length > 0 && (
                                    <TableWidget
                                        entityType={entityTypes.CONTROL}
                                        header={`${
                                            failedComplianceResults.length
                                        } controls failed across this node`}
                                        rows={failedComplianceResults}
                                        noDataText="No Controls"
                                        className="bg-base-100"
                                        columns={entityToColumns[entityTypes.CONTROL]}
                                        idAttribute="control.id"
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

Node.propTypes = entityComponentPropTypes;
Node.defaultProps = entityComponentDefaultProps;

export default Node;
