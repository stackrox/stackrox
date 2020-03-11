import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { Link } from 'react-router-dom';
import useCases from 'constants/useCaseTypes';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import SeverityLabel from 'Components/SeverityLabel';
import LifecycleStageLabel from 'Components/LifecycleStageLabel';
import getSubListFromEntity from 'utils/getSubListFromEntity';
import Widget from 'Components/Widget';
import Metadata from 'Components/Metadata';
import Button from 'Components/Button';
import isGQLLoading from 'utils/gqlLoading';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import EntityList from '../../List/EntityList';
import PolicyFindings from './PolicyFindings';

const PolicyEditButton = ({ id }) => {
    return (
        <Link className="no-underline text-base-600 mx-4" to={`/main/policies/${id}`}>
            <Button className="btn btn-base" text="Edit Policy" />
        </Link>
    );
};

PolicyEditButton.propTypes = {
    id: PropTypes.string.isRequired
};

const Policy = ({ id, entityListType, entityId1, query, entityContext }) => {
    const searchParam = useContext(searchContext);
    const variables = {
        cacheBuster: new Date().getUTCMilliseconds(),
        id,
        query: queryService.objectToWhereClause({
            ...query[searchParam],
            'Policy Id': id,
            'Lifecycle Stage': 'DEPLOY'
        })
    };

    const defaultQuery = gql`
        query getPolicy($id: ID!) {
            policy(id: $id) {
                id
                description
                lifecycleStages
                categories
                disabled
                enforcementActions
                rationale
                remediation
                severity
                whitelists {
                    name
                }
                deploymentCount
                alerts {
                    id
                    deployment {
                        id
                        name
                        clusterName
                        namespace
                    }
                    enforcement {
                        action
                        message
                    }
                    policy {
                        id
                        severity
                    }
                    time
                }
            }
        }
    `;

    function getQuery() {
        if (!entityListType) return defaultQuery;
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityTypes.POLICY,
            entityListType,
            useCases.CONFIG_MANAGEMENT
        );

        return gql`
            query getPolicy_${entityListType}($id: ID!, $query: String) {
                policy(id: $id) {
                    id
                    ${listFieldName}{ ...${fragmentName} }
                }
            }
            ${fragment}
        `;
    }

    return (
        <Query query={getQuery()} variables={variables}>
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) return <Loader />;
                const { policy: entity } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.POLICY} />;

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={getSubListFromEntity(entity, entityListType)}
                            query={query}
                            entityContext={{ ...entityContext, [entityTypes.POLICY]: id }}
                        />
                    );
                }
                const {
                    lifecycleStages = [],
                    categories = [],
                    severity = '',
                    description = '',
                    rationale,
                    remediation,
                    disabled,
                    enforcementActions,
                    whitelists = [],
                    alerts = [],
                    deploymentCount
                } = entity;

                const metadataKeyValuePairs = [
                    {
                        key: 'Life Cycle',
                        value: lifecycleStages.map(lifecycleStage => (
                            <LifecycleStageLabel
                                key={lifecycleStage}
                                lifecycleStage={lifecycleStage}
                            />
                        ))
                    },
                    {
                        key: 'Severity',
                        value: <SeverityLabel severity={severity} />
                    },
                    {
                        key: 'Enforced',
                        value: enforcementActions ? 'Yes' : 'No'
                    },
                    {
                        key: 'Enabled',
                        value: !disabled ? 'Yes' : 'No'
                    }
                ];

                const alertsData = alerts.reduce((acc, curr) => {
                    const datum = {
                        time: curr.time,
                        ...curr.deployment
                    };
                    return [...acc, datum];
                }, []);

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection
                            title="Policy Summary"
                            headerComponents={<PolicyEditButton id={id} />}
                        >
                            <div className="grid grid-gap-6 grid-columns-4 mx-4 grid-dense mb-4 pdf-page">
                                <Metadata
                                    className="sx-2 bg-base-100 min-h-48 h-full"
                                    keyValuePairs={metadataKeyValuePairs}
                                    whitelists={whitelists}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 h-full mb-4"
                                    name="Deployments"
                                    value={deploymentCount}
                                    entityType={entityTypes.DEPLOYMENT}
                                />
                                <Widget
                                    className="sx-1 min-h-48 h-full"
                                    bodyClassName="leading-normal p-4"
                                    header="Categories"
                                >
                                    {categories.join(', ')}
                                </Widget>
                                <Widget
                                    className="sx-1 min-h-48 h-full"
                                    bodyClassName="leading-normal p-4"
                                    header="Description"
                                >
                                    {description}
                                </Widget>
                                <Widget
                                    className="sx-2 min-h-48 h-full"
                                    bodyClassName="leading-normal"
                                    header="Remediation"
                                >
                                    <div className="p-4 border-r border-base-300">
                                        {remediation}
                                    </div>
                                    <div className="p-4">
                                        <span className="font-700">Rationale:&nbsp;</span>
                                        <span className="italic">{rationale}</span>
                                    </div>
                                </Widget>
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Policy Findings">
                            <div className="flex mb-4 pdf-page pdf-stretch">
                                <PolicyFindings
                                    entityContext={entityContext}
                                    policyId={id}
                                    alerts={alertsData}
                                />
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};

Policy.propTypes = entityComponentPropTypes;
Policy.defaultProps = entityComponentDefaultProps;

export default Policy;
