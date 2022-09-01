import React, { useContext, useState } from 'react';
import { format } from 'date-fns';
import { Formik } from 'formik';
import pluralize from 'pluralize';
import { Power, Edit } from 'react-feather';

import ButtonLink from 'Components/ButtonLink';
import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import PanelButton from 'Components/PanelButton';
import SeverityLabel from 'Components/SeverityLabel';
import StatusChip from 'Components/StatusChip';
import ToggleSwitch from 'Components/ToggleSwitch';
import Widget from 'Components/Widget';
import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import ViolationsAcrossThisDeployment from 'Containers/Workflow/widgets/ViolationsAcrossThisDeployment';
import { getCurriedDeploymentTableColumns } from 'Containers/VulnMgmt/List/Deployments/VulnMgmtListDeployments';
import { updatePolicyDisabledState } from 'services/PoliciesService';
import { entityGridContainerBaseClassName } from 'Containers/Workflow/WorkflowEntityPage';
import BooleanPolicySection from 'Containers/Policies/Wizard/Step3/BooleanPolicyLogicSection';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { getExcludedNamesByType } from 'utils/policyUtils';
import { pluralizeHas } from 'utils/textUtils';
import { getClientWizardPolicy } from 'Containers/Policies/policies.utils';
import MitreAttackVectors from 'Containers/MitreAttackVectors';
import { FormSection, FormSectionBody } from './FormSection';
import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidget from '../TableWidget';

const emptyPolicy = {
    categories: [],
    deploymentCount: 0,
    deployments: [],
    description: '',
    disabled: true,
    enforcementActions: [],
    fields: {},
    id: '',
    lastUpdated: '',
    latestViolation: '',
    lifecycleStages: [],
    name: '',
    policySections: [],
    policyStatus: '',
    rationale: '',
    remediation: '',
    scope: [],
    severity: '',
    exclusions: [],
};

const noop = () => {};
const VulnMgmtPolicyOverview = ({ data, entityContext, setRefreshTrigger }) => {
    const workflowState = useContext(workflowStateContext);
    const { isFeatureFlagEnabled } = useFeatureFlags();

    // guard against incomplete GraphQL-cached data
    const safeData = { ...emptyPolicy, ...data };

    const {
        id,
        name,
        policyStatus,
        description,
        disabled,
        rationale,
        remediation,
        severity,
        categories,
        latestViolation,
        lastUpdated,
        enforcementActions,
        lifecycleStages,
        policySections,
        scope,
        exclusions,
        deployments,
    } = safeData;
    const [currentDisabledState, setCurrentDisabledState] = useState(disabled);

    const initialValues = getClientWizardPolicy(safeData);

    function togglePolicy() {
        updatePolicyDisabledState(id, !currentDisabledState).then(() => {
            setCurrentDisabledState(!currentDisabledState);
            if (typeof setRefreshTrigger === 'function') {
                setRefreshTrigger(Math.random());
            }
        });
    }
    const policyActionButtons = (
        <div className="flex px-4">
            <PanelButton
                icon={<Power className="h-4 w-4 xl:ml-1" />}
                className={`btn ml-2 ${currentDisabledState ? 'btn-base' : 'btn-success'}`}
                onClick={togglePolicy}
                tooltip={`${currentDisabledState ? 'Toggle Policy On' : 'Toggle Policy Off'}`}
            >
                <label
                    htmlFor="enableDisablePolicy"
                    className={`block leading-none ${
                        currentDisabledState ? 'text-primary-700' : 'text-success-600'
                    } font-600 text-sm`}
                >
                    Policy
                </label>
                <ToggleSwitch
                    extraClassNames="mt-1"
                    id="enableDisablePolicy"
                    name="enableDisablePolicy"
                    toggleHandler={noop}
                    enabled={!currentDisabledState}
                    small
                />
            </PanelButton>
            <ButtonLink linkTo={`/main/policies/${id}/edit`} extraClassNames="ml-2">
                <span className="mr-2">Edit</span>
                <Edit size="16" />
            </ButtonLink>
        </div>
    );

    // @TODO: extract this out to make it re-usable and easier to test
    const failingDeployments = deployments.filter((singleDeploy) => {
        if (singleDeploy.policyStatus === 'pass') {
            return false;
        }
        return singleDeploy.deployAlerts.some((alert) => {
            return alert && alert.policy && alert.policy.id === id;
        });
    });

    const descriptionBlockMetadata = [
        {
            key: 'Description',
            value: description || '-',
        },
        {
            key: 'Rationale',
            value: rationale || '-',
        },
        {
            key: 'Remediation',
            value: remediation || '-',
        },
    ];

    const details = [
        {
            key: 'Categories',
            value: categories && categories.join(', '),
        },
        {
            key: 'Last violated',
            value: latestViolation ? format(latestViolation, dateTimeFormat) : '-',
        },
        {
            key: 'Last updated',
            value: lastUpdated ? format(lastUpdated, dateTimeFormat) : '-',
        },
        {
            key: 'Enforcement',
            value: enforcementActions && enforcementActions.length ? 'Yes' : 'No',
        },
        {
            key: 'Lifecycle',
            value:
                lifecycleStages && lifecycleStages.length
                    ? lifecycleStages.map((stage) => stage.toLowerCase()).join(', ')
                    : 'No lifecycle stages',
        },
    ];

    const scopeDetails = [
        {
            key: 'Cluster',
            value: (scope && scope.cluster) || 'N/A',
        },
        {
            key: 'Namespace',
            value: (scope && scope.namespace) || 'N/A',
        },
    ];

    const excludedScopesDetails = [
        {
            key: 'Image(s)',
            value: getExcludedNamesByType(exclusions, 'image') || 'N/A',
        },
        {
            key: 'Deployment(s)',
            value: getExcludedNamesByType(exclusions, 'deployment') || 'N/A',
        },
    ];

    const newEntityContext = { ...entityContext, [entityTypes.POLICY]: id };

    // need to know if this policy is scoped to a deployment higher up in the hierarchy
    const deploymentAncestor = workflowState.getSingleAncestorOfType(entityTypes.DEPLOYMENT);

    let policyFindingsContent = null;
    if (deploymentAncestor) {
        policyFindingsContent = (
            <ViolationsAcrossThisDeployment
                deploymentID={entityContext[entityTypes.DEPLOYMENT] || deploymentAncestor.entityId}
                policyID={id}
                message="No policies failed across this deployment"
            />
        );
    } else {
        const getDeploymentTableColumns = getCurriedDeploymentTableColumns(isFeatureFlagEnabled);

        policyFindingsContent = (
            <div className="pdf-page pdf-stretch pdf-new flex shadow rounded relativebg-base-100 mb-4 mx-4">
                <TableWidget
                    header={`${failingDeployments.length} ${pluralize(
                        entityTypes.DEPLOYMENT,
                        failingDeployments.length
                    )} ${pluralizeHas(failingDeployments.length)} failed across this policy`}
                    rows={failingDeployments}
                    entityType={entityTypes.DEPLOYMENT}
                    noDataText="No deployments have failed across this policy"
                    className="bg-base-100"
                    columns={getDeploymentTableColumns(workflowState)}
                    defaultSorted={[
                        {
                            id: 'priority',
                            desc: false,
                        },
                    ]}
                />
            </div>
        );
    }

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow min-w-0">
                <div className="grid grid-cols-3">
                    <div className="border-b border-base-300 col-span-3 flex flex-1 justify-end p-3">
                        {policyActionButtons}
                    </div>
                    <div className="col-span-2 border-r border-base-300">
                        <CollapsibleSection
                            title="Policy Summary"
                            // headerComponents={policyActionButtons}
                        >
                            {/* using a different container class here because we want default 3 columns instead of 2 */}
                            <div
                                className={`${entityGridContainerBaseClassName} grid-columns-2 lg:grid-columns-3`}
                            >
                                <div className="sx-3">
                                    <Widget
                                        header="Description, Rationale, & Remediation"
                                        className="bg-base-100 min-h-48 w-full h-full pdf-page pdf-stretch"
                                    >
                                        <div className="flex flex-col w-full">
                                            <div className="w-full bg-primary-200 text-2xl text-base-500 flex flex-col md:flex-row items-start md:items-center justify-between mb-2">
                                                <div className="w-full flex-grow p-4">
                                                    <span>{name}</span>
                                                </div>
                                                <div className="w-full flex border-t border-base-400 md:border-t-0 justify-end items-center">
                                                    <span className="flex flex-col items-center text-center px-4 py-4 border-base-400 border-l">
                                                        <span className="mb-2 text-xl">
                                                            Severity:
                                                        </span>
                                                        <SeverityLabel severity={severity} />
                                                    </span>
                                                    <span className="flex flex-col items-center text-center px-4 py-4 border-base-400 border-l">
                                                        <span className="mb-2 text-xl">
                                                            Status:
                                                        </span>
                                                        <StatusChip status={policyStatus} />
                                                    </span>
                                                </div>
                                            </div>
                                            <ul className="w-full flex-1 border-base-300">
                                                {descriptionBlockMetadata.map(
                                                    ({ key, value }, index) => (
                                                        <li
                                                            className={`${
                                                                index ===
                                                                descriptionBlockMetadata.length - 1
                                                                    ? ''
                                                                    : 'border-b'
                                                            } border-base-300 px-4 py-2 leading-normal`}
                                                            key={key}
                                                        >
                                                            <span className="text-base-700 mr-2 font-700">
                                                                {key}:
                                                            </span>
                                                            {value}
                                                        </li>
                                                    )
                                                )}
                                            </ul>
                                        </div>
                                    </Widget>
                                </div>
                                <div className="sx-2 sy-2">
                                    <Metadata
                                        className="h-full w-full min-w-43 bg-base-100 pdf-page"
                                        keyValuePairs={details}
                                        title="Details"
                                    />
                                </div>
                                <div className="sx-1 sy-1">
                                    <Metadata
                                        className="flex-1 bg-base-100 min-h-43 pdf-page h-full"
                                        keyValuePairs={scopeDetails}
                                        title="Scope"
                                    />
                                </div>
                                <div className="sx-1 sy-1">
                                    <Metadata
                                        className="flex-1 bg-base-100 min-h-43 pdf-page h-full"
                                        keyValuePairs={excludedScopesDetails}
                                        title="Excluded Scopes"
                                    />
                                </div>
                            </div>
                        </CollapsibleSection>
                        {!!id && (
                            <CollapsibleSection
                                title="MITRE ATT&CK"
                                dataTestId="mitre-attack-section"
                            >
                                <div className="p-4">
                                    <FormSection dataTestId="mitreAttackVectorDetails">
                                        <FormSectionBody>
                                            <MitreAttackVectors policyId={id} />
                                        </FormSectionBody>
                                    </FormSection>
                                </div>
                            </CollapsibleSection>
                        )}
                    </div>
                    <div className="col-span-1 border-b border-base-300">
                        {!!policySections.length && (
                            <div className="p-4 mt-10">
                                <Widget
                                    header="Policy Criteria"
                                    className="pdf-page pdf-stretch h-full"
                                >
                                    <Formik initialValues={initialValues} onSubmit={() => {}}>
                                        {() => <BooleanPolicySection readOnly />}
                                    </Formik>
                                </Widget>
                            </div>
                        )}
                    </div>
                    <div className="col-span-3 w-full">
                        <CollapsibleSection
                            title="Policy Findings"
                            dataTestId="policy-findings-section"
                        >
                            {policyFindingsContent}
                        </CollapsibleSection>
                    </div>
                </div>
            </div>
            <RelatedEntitiesSideList
                entityType={entityTypes.POLICY}
                entityContext={newEntityContext}
                data={safeData}
            />
        </div>
    );
};

export default VulnMgmtPolicyOverview;
