import React, { useContext } from 'react';
import { Link } from 'react-router-dom';
import { gql } from '@apollo/client';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import PolicySeverityIconText from 'Components/PatternFly/IconText/PolicySeverityIconText';
import Widget from 'Components/Widget';
import Metadata from 'Components/Metadata';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import entityTypes from 'constants/entityTypes';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import searchContext from 'Containers/searchContext';
import { formatLifecycleStages } from 'Containers/Policies/policies.utils';
import useIsRouteEnabled from 'hooks/usePermissions';
import getSubListFromEntity from 'utils/getSubListFromEntity';
import isGQLLoading from 'utils/gqlLoading';
import queryService from 'utils/queryService';
import { policiesBasePath } from 'routePaths';

import { getConfigMgmtCountQuery } from '../../ConfigMgmt.utils';
import EntityList from '../../List/EntityList';
import PolicyFindings from './PolicyFindings';

const Policy = ({ id, entityListType, entityId1, query, entityContext, pagination }) => {
    const isRouteEnabled = useIsRouteEnabled();

    const isRouteEnabledForPolicy = isRouteEnabled('policy-management');

    const searchParam = useContext(searchContext);
    const variables = {
        id,
        query: queryService.objectToWhereClause({
            ...query[searchParam],
            'Policy Id': id,
            'Lifecycle Stage': 'DEPLOY',
        }),
        pagination,
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
                exclusions {
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
        if (!entityListType) {
            return defaultQuery;
        }
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityTypes.POLICY,
            entityListType,
            useCases.CONFIG_MANAGEMENT
        );
        const countQuery = getConfigMgmtCountQuery(entityListType);

        return gql`
            query getPolicy_${entityListType}($id: ID!, $query: String, $pagination: Pagination) {
                policy(id: $id) {
                    id
                    ${listFieldName}(query: $query, pagination: $pagination){ ...${fragmentName} }
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
                const { policy: entity } = data;
                if (!entity) {
                    return (
                        <PageNotFound
                            resourceType={entityTypes.POLICY}
                            useCase={useCases.CONFIG_MANAGEMENT}
                        />
                    );
                }

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            entityId={entityId1}
                            data={getSubListFromEntity(entity, entityListType)}
                            totalResults={data?.policy?.count}
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
                    exclusions = [],
                    alerts = [],
                    deploymentCount,
                } = entity;

                const metadataKeyValuePairs = [
                    {
                        key: 'Lifecycle Stage',
                        value: formatLifecycleStages(lifecycleStages),
                    },
                    {
                        key: 'Severity',
                        value: <PolicySeverityIconText severity={severity} isTextOnly={false} />,
                    },
                    {
                        key: 'Enforced',
                        value: enforcementActions ? 'Yes' : 'No',
                    },
                    {
                        key: 'Enabled',
                        value: !disabled ? 'Yes' : 'No',
                    },
                ];

                const alertsData = alerts.reduce((acc, curr) => {
                    const datum = {
                        time: curr.time,
                        ...curr.deployment,
                    };
                    return [...acc, datum];
                }, []);

                const headerComponents = isRouteEnabledForPolicy ? (
                    <Link
                        className="no-underline text-base-600 mx-4 btn btn-base"
                        to={`${policiesBasePath}/${id}`}
                    >
                        View policy
                    </Link>
                ) : null;

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection
                            title="Policy Summary"
                            headerComponents={headerComponents}
                        >
                            <div className="grid grid-gap-6 grid-columns-4 mx-4 grid-dense mb-4 pdf-page">
                                <Metadata
                                    className="sx-2 bg-base-100 min-h-48 h-full"
                                    keyValuePairs={metadataKeyValuePairs}
                                    exclusions={exclusions}
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
                                        <span>{rationale}</span>
                                    </div>
                                </Widget>
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection
                            title="Policy Findings"
                            dataTestId="policy-findings-section"
                        >
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
