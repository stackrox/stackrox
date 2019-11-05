import React, { useContext } from 'react';
import { format } from 'date-fns';
import pluralize from 'pluralize';

import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import SeverityLabel from 'Components/SeverityLabel';
import StatusChip from 'Components/StatusChip';
import Widget from 'Components/Widget';
import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import { getDeploymentTableColumns } from 'Containers/VulnMgmt/List/Deployments/VulnMgmtListDeployments';
import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidget from '../TableWidget';
import PolicyConfigurationFields from './PolicyConfigurationFields';

const VulnMgmtPolicyOverview = ({ data, entityContext }) => {
    const workflowState = useContext(workflowStateContext);

    const {
        id,
        name,
        policyStatus,
        description,
        rationale,
        remediation,
        severity,
        categories,
        latestViolation,
        lastUpdated,
        enforcementActions,
        lifecycleStages,
        fields,
        deploymentCount,
        scope,
        whitelists,
        deployments
    } = data;

    // @TODO: extract this out to make it re-usable and easier to test
    const failingDeployments = deployments.filter(singleDeploy => {
        if (
            singleDeploy.policyStatus === 'pass' ||
            !singleDeploy.deployAlerts ||
            !singleDeploy.deployAlerts.length
        ) {
            return false;
        }
        return singleDeploy.deployAlerts.some(alert => {
            return alert && alert.policy && alert.policy.id === id;
        });
    });

    const drrMetadata = [
        {
            key: 'Description',
            value: description || '-'
        },
        {
            key: 'Rationale',
            value: rationale || '-'
        },
        {
            key: 'Remediation',
            value: remediation || '-'
        }
    ];

    const details = [
        {
            key: 'Categories',
            value: categories && categories.join(', ')
        },
        {
            key: 'Last violated',
            value: format(latestViolation, dateTimeFormat)
        },
        {
            key: 'Last updated',
            value: format(lastUpdated, dateTimeFormat)
        },
        {
            key: 'Enforcement',
            value: enforcementActions || enforcementActions.length ? 'Yes' : 'No'
        },
        {
            key: 'Lifecycle',
            value:
                lifecycleStages && lifecycleStages.length
                    ? lifecycleStages.map(stage => stage.toLowerCase()).join(', ')
                    : 'No lifecycle stages'
        }
    ];

    const scopeDetails = [
        {
            key: 'Cluster',
            value: (scope && scope.cluster) || 'N/A'
        },
        {
            key: 'Namespace',
            value: (scope && scope.namespace) || 'N/A'
        }
    ];

    const whitelistDetails = [
        {
            key: 'Image(s)',
            value: (whitelists && whitelists.image && whitelists.image.name) || 'N/A'
        },
        {
            key: 'Deployment(s)',
            value: (whitelists && whitelists.image && whitelists.image.name) || 'N/A'
        }
    ];

    function getCountData(entityType) {
        switch (entityType) {
            case entityTypes.DEPLOYMENT:
                return deploymentCount;
            default:
                return 0;
        }
    }

    const newEntityContext = { ...entityContext, [entityTypes.POLICY]: id };

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow">
                <CollapsibleSection title="Policy summary">
                    <div className="flex mb-4 pdf-page">
                        <Widget
                            header="Description, Rationale, & Remdediation"
                            headerComponents={null}
                            className="ml-4 mr-2 bg-base-100 min-h-48 mb-4 w-2/3"
                        >
                            <div className="flex flex-col w-full">
                                <div className="bg-primary-200 text-2xl text-base-500 flex flex-col xl:flex-row items-start xl:items-center justify-between">
                                    <div className="w-full flex-grow p-4">
                                        <span>{name}</span>
                                    </div>
                                    <div className="w-full flex border-t border-base-400 xl:border-t-0 justify-end items-center">
                                        <span className="flex flex-col items-center text-center px-4 py-4 border-base-400 border-l">
                                            <span>Severity:</span>
                                            <SeverityLabel severity={severity} />
                                        </span>
                                        <span className="flex flex-col items-center text-center px-4 py-4 border-base-400 border-l">
                                            <span>Status:</span>
                                            <StatusChip status={policyStatus} />
                                        </span>
                                    </div>
                                </div>
                                <ul className="flex-1 list-reset border-r border-base-300">
                                    {drrMetadata.map(({ key, value }) => (
                                        <li
                                            className="border-b border-base-300 px-4 py-2"
                                            key={key}
                                        >
                                            <span className="text-base-700 font-600 mr-2">
                                                {key}:
                                            </span>
                                            {value}
                                        </li>
                                    ))}
                                </ul>
                            </div>
                        </Widget>
                        <Metadata
                            className="w-1/3 mx-2 min-w-48 bg-base-100 min-h-48 mb-4"
                            keyValuePairs={details}
                            title="Details"
                        />
                    </div>
                    <div className="flex mb-4 pdf-page">
                        <PolicyConfigurationFields
                            className="flex-1 mx-2 min-w-48 bg-base-100 h-48 mb-4"
                            fields={fields}
                        />
                        <Metadata
                            className="flex-1 mx-2 min-w-48 bg-base-100 h-48 mb-4"
                            keyValuePairs={scopeDetails}
                            title="Scope"
                        />
                        <Metadata
                            className="flex-1 mx-2 min-w-48 bg-base-100 h-48 mb-4"
                            keyValuePairs={whitelistDetails}
                            title="Whitelist"
                        />
                    </div>
                </CollapsibleSection>
                <CollapsibleSection title="Policy Findings">
                    <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                        <TableWidget
                            header={`${failingDeployments.length} ${pluralize(
                                entityTypes.DEPLOYMENT,
                                failingDeployments.length
                            )} have failed across this policy`}
                            rows={failingDeployments}
                            entityType={entityTypes.DEPLOYMENT}
                            noDataText="No deployments have failed across this policy"
                            className="bg-base-100"
                            columns={getDeploymentTableColumns(workflowState, false)}
                            idAttribute="cve"
                        />
                    </div>
                </CollapsibleSection>
            </div>
            <RelatedEntitiesSideList
                entityType={entityTypes.POLICY}
                workflowState={workflowState}
                getCountData={getCountData}
                entityContext={newEntityContext}
            />
        </div>
    );
};

export default VulnMgmtPolicyOverview;
