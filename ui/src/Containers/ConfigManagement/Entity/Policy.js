import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { entityViolationsColumns } from 'constants/listColumns';
import { Link } from 'react-router-dom';
import { DEPLOYMENT_FRAGMENT } from 'queries/deployment';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import SeverityLabel from 'Components/SeverityLabel';
import LifecycleStageLabel from 'Components/LifecycleStageLabel';
import Widget from 'Components/Widget';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import TableWidget from 'Containers/ConfigManagement/Entity/widgets/TableWidget';
import Button from 'Components/Button';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import EntityList from '../List/EntityList';
import getSubListFromEntity from '../List/utilities/getSubListFromEntity';

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

const DeploymentViolations = ({ className, alerts }) => {
    const rows = alerts;
    const columns = entityViolationsColumns[entityTypes.DEPLOYMENT];
    return (
        <TableWidget
            header={`${rows.length} Deployments in Violation`}
            entityType={entityTypes.DEPLOYMENT}
            columns={columns}
            rows={rows}
            idAttribute="deployment.id"
            noDataText="No Deployments in Violation"
            className={className}
        />
    );
};

DeploymentViolations.propTypes = {
    className: PropTypes.string,
    alerts: PropTypes.arrayOf(PropTypes.shape({}))
};

DeploymentViolations.defaultProps = {
    className: '',
    alerts: []
};

const Policy = ({ id, entityListType, query }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        where: queryService.objectToWhereClause(query[searchParam])
    };

    const QUERY = gql`
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
            deployments {
                ${entityListType === entityTypes.DEPLOYMENT ? '...deploymentFields' : 'id'}
            }
            alerts {
                id
                deployment {
                    id
                    name
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
    ${entityListType === entityTypes.DEPLOYMENT ? DEPLOYMENT_FRAGMENT : ''}

`;

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                const { policy: entity } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.POLICY} />;

                const {
                    id: policyId,
                    lifecycleStages = [],
                    categories = [],
                    severity,
                    description,
                    rationale,
                    remediation,
                    disabled,
                    enforcementActions,
                    whitelists = [],
                    alerts = [],
                    deployments = []
                } = entity;

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            data={getSubListFromEntity(entity, entityListType)}
                            query={query}
                        />
                    );
                }

                const metadataKeyValuePairs = [
                    {
                        key: 'ID',
                        value: policyId
                    },
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

                return (
                    <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection
                            title="Policy Details"
                            headerComponents={<PolicyEditButton id={id} />}
                        >
                            <div className="grid grid-gap-6 grid-columns-4 mx-4 grid-dense mb-4 pdf-page">
                                <Metadata
                                    className="sx-2 bg-base-100 h-48"
                                    keyValuePairs={metadataKeyValuePairs}
                                    whitelists={whitelists}
                                />
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    name="Deployments"
                                    value={deployments.length}
                                    entityType={entityTypes.DEPLOYMENT}
                                />
                                <Widget
                                    className="sx-1 h-48"
                                    bodyClassName="leading-normal p-4"
                                    header="Categories"
                                >
                                    {categories.join(', ')}
                                </Widget>
                                <Widget
                                    className="sx-1 h-48"
                                    bodyClassName="leading-normal p-4"
                                    header="Description"
                                >
                                    {description}
                                </Widget>
                                <Widget
                                    className="sx-2 h-48"
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
                        {!!alerts.length && (
                            <CollapsibleSection title="Policy Findings">
                                <div className="flex mb-4 pdf-page pdf-stretch p-4">
                                    <DeploymentViolations
                                        className="mx-4 w-full bg-base-100"
                                        alerts={alerts}
                                    />
                                </div>
                            </CollapsibleSection>
                        )}
                    </div>
                );
            }}
        </Query>
    );
};

Policy.propTypes = entityComponentPropTypes;
Policy.defaultProps = entityComponentDefaultProps;

export default Policy;
