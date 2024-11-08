import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { format } from 'date-fns';
import pluralize from 'pluralize';
import { gql } from '@apollo/client';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Components/RelatedEntity';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import Metadata from 'Components/Metadata';
import CollapsibleRow from 'Components/CollapsibleRow';
import Widget from 'Components/Widget';
import dateTimeFormat from 'constants/dateTimeFormat';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import searchContext from 'Containers/searchContext';
import { getConfigMgmtCountQuery } from 'Containers/ConfigManagement/ConfigMgmt.utils';
import getSubListFromEntity from 'utils/getSubListFromEntity';
import isGQLLoading from 'utils/gqlLoading';
import queryService from 'utils/queryService';
import EntityList from '../List/EntityList';

const SecretDataMetadata = ({ metadata }) => {
    if (!metadata) {
        return null;
    }
    const { startDate, endDate, issuer = {}, sans = [], subject = {} } = metadata;
    const {
        commonName: issuerCommonName = 'N/A',
        names: issuerNames,
        organizationUnit = 'N/A',
    } = issuer;
    const { commonName: subjectCommonName = 'N/A', names: subjectNames } = subject;
    return (
        <div className="flex flex-row">
            <Widget
                header="Timeframe"
                className="m-4"
                bodyClassName="flex flex-col p-4 leading-normal"
            >
                <div>
                    <span className="font-700 mr-4">Start Date:</span>
                    <span>{startDate ? format(startDate, dateTimeFormat) : 'N/A'}</span>
                </div>
                <div>
                    <span className="font-700 mr-4">End Date:</span>
                    <span>{endDate ? format(endDate, dateTimeFormat) : 'N/A'}</span>
                </div>
            </Widget>
            <Widget
                header="Issuer"
                className="m-4"
                bodyClassName="flex flex-col p-4 leading-normal"
            >
                <div>
                    <span className="font-700 mr-4">Common Name:</span>
                    <span>{issuerCommonName}</span>
                </div>
                <div>
                    <span className="font-700 mr-4">Name(s):</span>
                    <span>{issuerNames ? issuerNames.join(', ') : 'None'}</span>
                </div>
                <div>
                    <span className="font-700 mr-4">Organization Unit:</span>
                    <span>{organizationUnit}</span>
                </div>
            </Widget>
            <Widget
                header="Subject"
                className="m-4"
                bodyClassName="flex flex-col p-4 leading-normal"
            >
                <div>
                    <span className="font-700 mr-4">Common Name:</span>
                    <span>{subjectCommonName}</span>
                </div>
                <div>
                    <span className="font-700 mr-4">Name(s):</span>
                    <span>{subjectNames ? subjectNames.join(', ') : 'None'}</span>
                </div>
            </Widget>
            {!!sans.length && (
                <Widget
                    header="SANS"
                    className="m-4"
                    bodyClassName="flex flex-col p-4 leading-normal"
                >
                    <div>
                        <span className="font-700 mr-4">SANS:</span>
                        <span>{sans.join(', ')}</span>
                    </div>
                </Widget>
            )}
        </div>
    );
};

SecretDataMetadata.propTypes = {
    metadata: PropTypes.shape(),
};

SecretDataMetadata.defaultProps = {
    metadata: null,
};

const SecretValues = ({ files }) => {
    const filesWithoutImagePullSecrets = files.filter(
        (file) =>
            !file.metadata || (file.metadata && file.metadata.__typename !== 'ImagePullSecret')
    );
    const filesCount = filesWithoutImagePullSecrets.length;
    const widgetHeader = `${filesCount} ${pluralize('value', filesCount)}`;
    const secretValues = filesWithoutImagePullSecrets.map((file) => {
        const { name, type, metadata } = file;
        const { algorithm } = metadata || {};
        const collapsibleRowHeader = (
            <div className="flex flex-1 w-full">
                <div className="flex flex-1">{name}</div>
                {type && (
                    <div className="border-l border-base-400 px-2 capitalize">
                        {type.replace(/_/g, ' ').toLowerCase()}
                    </div>
                )}
                {algorithm && <div className="border-l border-base-400 px-2">{algorithm}</div>}
            </div>
        );
        return (
            <CollapsibleRow key={name} header={collapsibleRowHeader} isCollapsible={!!metadata}>
                <SecretDataMetadata metadata={metadata} />
            </CollapsibleRow>
        );
    });
    return (
        <Widget header={widgetHeader} bodyClassName="flex flex-col">
            {secretValues}
        </Widget>
    );
};

SecretValues.propTypes = {
    files: PropTypes.arrayOf(PropTypes.shape).isRequired,
};

const Secret = ({ id, entityListType, entityId1, query, entityContext, pagination }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        query: queryService.objectToWhereClause({
            ...query[searchParam],
        }),
        pagination,
    };

    const defaultQuery = gql`
        query getSecret($id: ID!) {
            secret(id: $id) {
                id
                name
                createdAt
                files {
                    name
                    type
                    metadata {
                        __typename
                        ... on Cert {
                            endDate
                            startDate
                            algorithm
                            issuer {
                                commonName
                                names
                            }
                            subject {
                                commonName
                                names
                            }
                            sans
                        }
                        ... on ImagePullSecret {
                            registries {
                                name
                                username
                            }
                        }
                    }
                }
                namespace
                deploymentCount
                labels {
                    key
                    value
                }
                annotations {
                    key
                    value
                }
                ${entityContext[entityTypes.CLUSTER] ? '' : 'clusterId clusterName'}
            }
        }
    `;

    function getQuery() {
        if (!entityListType) {
            return defaultQuery;
        }
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityTypes.SECRET,
            entityListType,
            useCases.CONFIG_MANAGEMENT
        );
        const countQuery = getConfigMgmtCountQuery(entityListType);

        return gql`
            query getSecret_${entityListType}($id: ID!, $query: String, $pagination: Pagination) {
                secret(id: $id) {
                    id
                    ${listFieldName}(query: $query, pagination: $pagination) { ...${fragmentName} }
                    ${countQuery}
                }
            }
            ${fragment}
        `;
    }
    return (
        <Query query={getQuery()} variables={variables} fetchPolicy="network-only">
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) {
                    return <Loader />;
                }
                if (!data || !data.secret) {
                    return (
                        <PageNotFound
                            resourceType={entityTypes.SECRET}
                            useCase={useCases.CONFIG_MANAGEMENT}
                        />
                    );
                }
                const { secret } = data;
                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={getSubListFromEntity(secret, entityListType)}
                            totalResults={data?.secret?.count}
                            query={query}
                        />
                    );
                }

                const {
                    createdAt,
                    labels = [],
                    annotations = [],
                    deploymentCount,
                    clusterName,
                    clusterId,
                    files = [],
                } = secret;

                const metadataKeyValuePairs = [
                    {
                        key: 'Created',
                        value: createdAt ? format(createdAt, dateTimeFormat) : 'N/A',
                    },
                ];

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Secret Summary">
                            <div className="flex mb-4 flex-wrap pdf-page">
                                <Metadata
                                    className="mx-4 bg-base-100 min-h-48 mb-4"
                                    keyValuePairs={metadataKeyValuePairs}
                                    labels={labels}
                                    annotations={annotations}
                                />
                                {clusterName && (
                                    <RelatedEntity
                                        className="mx-4 min-w-48 min-h-48 mb-4"
                                        entityType={entityTypes.CLUSTER}
                                        name="Cluster"
                                        value={clusterName}
                                        entityId={clusterId}
                                    />
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Deployments"
                                    value={deploymentCount}
                                    entityType={entityTypes.DEPLOYMENT}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Secret Values">
                            <div className="flex pdf-page pdf-stretch mb-4 ml-4 mr-4">
                                <SecretValues files={files} />
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};

Secret.propTypes = entityComponentPropTypes;
Secret.defaultProps = entityComponentDefaultProps;

export default Secret;
