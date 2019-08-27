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
import EntityList from '../List/EntityList';

const Node = ({ id, entityListType, query }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        query: queryService.objectToWhereClause(query[searchParam])
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
                complianceResults {
                    resource {
                        __typename
                    }
                    control {
                        id
                        standardId
                        name
                        description
                    }
                    value {
                        overallState
                        evidence {
                            message
                        }
                    }
                }
                controls(query: $query) {
                    id
                    standardId
                    name
                    description
                }
            }
        }
    `;
    // TODO: use passingControls and failingControls
    const NODES_QUERY = gql`
        query getNodesForControls($id: ID!, $clusterId: ID!) {
            node(id: $id) {
                id
                controls {
                    id
                    complianceControlNodes(clusterID: $clusterId) {
                        id
                        name
                    }
                }
            }
        }
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
                    complianceResults = [],
                    controls
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

                const failedComplianceResults = complianceResults
                    .filter(cr => cr.value.overallState === 'COMPLIANCE_STATE_FAILURE')
                    .map(cr => ({
                        ...cr,
                        control: {
                            ...cr.control,
                            standard: standardLabels[cr.control.standardId]
                        }
                    }));

                if (entityListType) {
                    return (
                        <Query query={NODES_QUERY} variables={{ id, clusterId }}>
                            {({ loading: nodesLoading, data: nodesData }) => {
                                if (nodesLoading) return <Loader />;
                                const { node: currentNode } = nodesData;
                                const { controls: controlList } = currentNode;
                                const controlMap = controlList.reduce(
                                    (acc, curr) => ({
                                        ...acc,
                                        [curr.id]: curr.complianceControlNodes.map(c => c.name)
                                    }),
                                    {}
                                );
                                const processedControls = controls.map(control => ({
                                    ...control,
                                    nodes: controlMap[control.id] || [],
                                    standard: standardLabels[control.standardId],
                                    control: `${control.name} - ${control.description}`,
                                    passing: !failedComplianceResults.find(
                                        cr => cr.control.id === control.id
                                    )
                                }));

                                return (
                                    <EntityList
                                        entityListType={entityListType}
                                        data={processedControls}
                                        query={query}
                                    />
                                );
                            }}
                        </Query>
                    );
                }

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
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Cluster"
                                    entityType={entityTypes.CLUSTER}
                                    value={clusterName}
                                    entityId={clusterId}
                                />
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
