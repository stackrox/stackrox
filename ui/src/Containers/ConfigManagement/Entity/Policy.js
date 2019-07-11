import React from 'react';
import PropTypes from 'prop-types';
import { POLICY as QUERY } from 'queries/policy';
import entityTypes from 'constants/entityTypes';
import { entityViolationsColumns } from 'constants/listColumns';
import { Link } from 'react-router-dom';

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
    alerts: PropTypes.arrayOf()
};

DeploymentViolations.defaultProps = {
    className: '',
    alerts: []
};

const Policy = ({ id }) => {
    return (
        <Query query={QUERY} variables={{ id }}>
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
                    labels = [],
                    annotations = [],
                    whitelists = [],
                    alerts = []
                } = entity;

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
                const metadataCounts = [
                    { value: labels.length, text: 'Labels' },
                    { value: annotations.length, text: 'Annotations' },
                    { value: whitelists.length, text: 'Whitelists' }
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
                                    counts={metadataCounts}
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

Policy.propTypes = {
    id: PropTypes.string.isRequired
};

export default Policy;
