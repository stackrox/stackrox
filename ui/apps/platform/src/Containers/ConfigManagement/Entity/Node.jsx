import React, { useContext } from 'react';
import { gql } from '@apollo/client';
import { format } from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import NoResultsMessage from 'Components/NoResultsMessage';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Components/RelatedEntity';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import Metadata from 'Components/Metadata';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import TableWidget from 'Containers/ConfigManagement/Entity/widgets/TableWidget';
import searchContext from 'Containers/searchContext';
import { standardLabels } from 'messages/standards';
import { CONTROL_FRAGMENT } from 'queries/controls';
import { sortVersion } from 'sorters/sorters';
import isGQLLoading from 'utils/gqlLoading';
import queryService from 'utils/queryService';
import getControlsWithStatus from '../List/utilities/getControlsWithStatus';
import EntityList from '../List/EntityList';

const Node = ({ id, entityListType, entityId1, query, entityContext, pagination }) => {
    const searchParam = useContext(searchContext);

    const queryObject = { ...query[searchParam] };
    if (!queryObject.Standard) {
        queryObject.Standard = 'CIS';
    }

    const variables = {
        id,
        query: queryService.getEntityWhereClause(queryObject),
        pagination,
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
                complianceResults(query: $query) {
                    ...controlFields
                }
            }
        }
        ${CONTROL_FRAGMENT}
    `;

    return (
        <Query query={QUERY} variables={variables} fetchPolicy="network-only">
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) {
                    return <Loader />;
                }
                if (!data || !data.node) {
                    return (
                        <PageNotFound
                            resourceType={entityTypes.NODE}
                            useCase={useCases.CONFIG_MANAGEMENT}
                        />
                    );
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
                    complianceResults = [],
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
                        value: joinedAt ? format(joinedAt, dateTimeFormat) : 'N/A',
                    },
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
                    .filter((cr) => cr.value.overallState === 'COMPLIANCE_STATE_FAILURE')
                    .map((cr) => ({
                        ...cr,
                        standard: standardLabels[cr.control.standardId],
                        controlName: `${cr.control.name} - ${cr.control.description}`,
                    }));

                const controlColumns = [
                    {
                        accessor: 'id',
                        Header: 'id',
                        headerClassName: 'hidden',
                        className: 'hidden',
                    },
                    {
                        accessor: 'standard',
                        sortMethod: sortVersion,
                        Header: 'Standard',
                        headerClassName: `w-1/5 ${defaultHeaderClassName}`,
                        className: `w-1/5 ${defaultColumnClassName}`,
                    },
                    {
                        accessor: 'controlName',
                        sortMethod: sortVersion,
                        Header: 'Control',
                        headerClassName: `w-1/2 ${defaultHeaderClassName}`,
                        className: `w-1/2 ${defaultColumnClassName}`,
                    },
                ];

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Node Summary">
                            <div className="flex mb-4 flex-wrap pdf-page">
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
                                        entityType={entityTypes.CLUSTER}
                                        value={clusterName}
                                        entityId={clusterId}
                                    />
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="CIS Controls"
                                    value={complianceResults.length}
                                    entityType={entityTypes.CONTROL}
                                />
                            </div>
                        </CollapsibleSection>
                        {!(entityContext && entityContext[entityTypes.CONTROL]) && (
                            <CollapsibleSection title="Node Findings">
                                <div className="flex pdf-page pdf-stretch shadow relative rounded bg-base-100 mb-4 ml-4 mr-4">
                                    {failedComplianceResults.length === 0 && (
                                        <NoResultsMessage
                                            message="No nodes failing controls on this node"
                                            className="p-3 shadow"
                                            icon="info"
                                        />
                                    )}
                                    {failedComplianceResults.length > 0 && (
                                        <TableWidget
                                            entityType={entityTypes.CONTROL}
                                            header={`${failedComplianceResults.length} controls failed across this node`}
                                            rows={failedComplianceResults}
                                            noDataText="No Controls"
                                            className="bg-base-100"
                                            columns={controlColumns}
                                            idAttribute="control.id"
                                            defaultSorted={[
                                                {
                                                    id: 'standard',
                                                    desc: false,
                                                },
                                                {
                                                    id: 'controlName',
                                                    desc: false,
                                                },
                                            ]}
                                        />
                                    )}
                                </div>
                            </CollapsibleSection>
                        )}
                    </div>
                );
            }}
        </Query>
    );
};

Node.propTypes = entityComponentPropTypes;
Node.defaultProps = entityComponentDefaultProps;

export default Node;
