import React, { ReactElement, useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import startCase from 'lodash/startCase';
import {
    Bullseye,
    Divider,
    Label,
    LabelGroup,
    PageSection,
    Spinner,
    Tab,
    TabTitleText,
    Tabs,
    Title,
} from '@patternfly/react-core';

import PolicyDetailContent from 'Containers/Policies/Detail/PolicyDetailContent';
import { getClientWizardPolicy } from 'Containers/Policies/policies.utils';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';
import { fetchAlert } from 'services/AlertsService';
import { Alert, isDeploymentAlert, isResourceAlert } from 'types/alert.proto';
import { getDateTime } from 'utils/dateUtils';
import { VIOLATION_STATE_LABELS } from 'constants/violationStates';

import useFilteredWorkflowViewURLState from 'Components/FilteredWorkflowViewSelector/useFilteredWorkflowViewURLState';
import DeploymentTabWithReadAccessForDeployment from './Deployment/DeploymentTabWithReadAccessForDeployment';
import DeploymentTabWithoutReadAccessForDeployment from './Deployment/DeploymentTabWithoutReadAccessForDeployment';
import NetworkPolicies from './NetworkPolicies/NetworkPoliciesTab';
import EnforcementDetails from './EnforcementDetails';
import ViolationNotFoundPage from '../ViolationNotFoundPage';
import ViolationDetails from './ViolationDetails';
import ViolationsBreadcrumbs from '../ViolationsBreadcrumbs';

function ViolationDetailsPage(): ReactElement {
    const isRouteEnabled = useIsRouteEnabled();
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForDeployment = hasReadAccess('Deployment');
    const hasReadAccessForNetworkPolicy = hasReadAccess('NetworkPolicy');
    const isRouteEnabledForPolicy = isRouteEnabled('policy-management');

    const [activeTabKey, setActiveTabKey] = useState(0);
    const [alert, setAlert] = useState<Alert | null>(null);
    const [isFetchingSelectedAlert, setIsFetchingSelectedAlert] = useState(false);

    const { alertId } = useParams() as { alertId: string };

    const { filteredWorkflowView } = useFilteredWorkflowViewURLState('Full view');

    function handleTabClick(_, tabIndex) {
        setActiveTabKey(tabIndex);
    }

    // Make updates to the fetching state, and selected alert.
    useEffect(() => {
        setIsFetchingSelectedAlert(true);
        fetchAlert(alertId).then(
            (result) => {
                setAlert(result);
                setIsFetchingSelectedAlert(false);
            },
            () => {
                setAlert(null);
                setIsFetchingSelectedAlert(false);
            }
        );
    }, [alertId, setAlert, setIsFetchingSelectedAlert]);

    if (!alert) {
        if (isFetchingSelectedAlert) {
            return (
                <Bullseye>
                    <Spinner />
                </Bullseye>
            );
        }
        return <ViolationNotFoundPage />;
    }

    const { policy, enforcement } = alert;
    const title = policy.name || 'Unknown violation';

    const entityName = isResourceAlert(alert)
        ? alert.resource.clusterName
        : isDeploymentAlert(alert)
          ? alert.deployment.name
          : '';

    const resourceType = isResourceAlert(alert) ? alert.resource.resourceType : 'deployment';

    const displayedResourceType = startCase(resourceType.toLowerCase());

    return (
        <>
            <ViolationsBreadcrumbs current={title} filteredWorkflowView={filteredWorkflowView} />
            <PageSection variant="light">
                <Title headingLevel="h1">{title}</Title>
                <Title
                    headingLevel="h2"
                    className="pf-v5-u-mb-sm"
                >{`in "${entityName}" ${displayedResourceType}`}</Title>
                <LabelGroup aria-label="Violation state and resolution">
                    <Label>State: {VIOLATION_STATE_LABELS[alert.state]}</Label>
                    <Label>Stte: {VIOLATION_STATE_LABELS[alert.state]}</Label>
                    <Label>Sate: {VIOLATION_STATE_LABELS[alert.state]}</Label>
                    <Label>State: {VIOLATION_STATE_LABELS[alert.state]}</Label>
                    <Label>State: {VIOLATION_STATE_LABELS[alert.state]}</Label>
                    <Label>State: {VIOLATION_STATE_LABELS[alert.state]}</Label>
                    {alert.state === 'RESOLVED' && (
                        <Label>
                            Resolved on:{' '}
                            {alert.resolvedAt ? getDateTime(alert.resolvedAt) : 'Not available'}
                        </Label>
                    )}
                </LabelGroup>
            </PageSection>
            <PageSection variant="default" padding={{ default: 'noPadding' }}>
                <Tabs
                    mountOnEnter
                    activeKey={activeTabKey}
                    onSelect={handleTabClick}
                    className="pf-v5-u-background-color-100 pf-v5-u-pl-lg"
                >
                    <Tab eventKey={0} title={<TabTitleText>Violation</TabTitleText>}>
                        <PageSection variant="default">
                            <ViolationDetails
                                violations={alert.violations}
                                processViolation={alert.processViolation}
                                lifecycleStage={alert.lifecycleStage}
                            />
                        </PageSection>
                    </Tab>
                    {enforcement && (
                        <Tab eventKey={1} title={<TabTitleText>Enforcement</TabTitleText>}>
                            <PageSection variant="default">
                                <EnforcementDetails alert={alert} enforcement={enforcement} />
                            </PageSection>
                        </Tab>
                    )}
                    {isDeploymentAlert(alert) && (
                        <Tab eventKey={2} title={<TabTitleText>Deployment</TabTitleText>}>
                            <PageSection variant="default">
                                {hasReadAccessForDeployment ? (
                                    <DeploymentTabWithReadAccessForDeployment
                                        alertDeployment={alert.deployment}
                                    />
                                ) : (
                                    <DeploymentTabWithoutReadAccessForDeployment
                                        alertDeployment={alert.deployment}
                                    />
                                )}
                            </PageSection>
                        </Tab>
                    )}
                    {isRouteEnabledForPolicy && (
                        <Tab eventKey={3} title={<TabTitleText>Policy</TabTitleText>}>
                            <PageSection variant="default">
                                <Title headingLevel="h3" className="pf-v5-u-mb-md">
                                    Policy overview
                                </Title>
                                <Divider component="div" className="pf-v5-u-pb-md" />
                                <PolicyDetailContent policy={getClientWizardPolicy(policy)} />
                            </PageSection>
                        </Tab>
                    )}
                    {isDeploymentAlert(alert) && hasReadAccessForNetworkPolicy && (
                        <Tab eventKey={4} title={<TabTitleText>Network policies</TabTitleText>}>
                            <PageSection variant="default">
                                <NetworkPolicies
                                    clusterId={alert.deployment.clusterId}
                                    namespaceName={alert.deployment.namespace}
                                />
                            </PageSection>
                        </Tab>
                    )}
                </Tabs>
            </PageSection>
        </>
    );
}

export default ViolationDetailsPage;
