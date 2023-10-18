import React, { ReactElement, useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import startCase from 'lodash/startCase';
import {
    TabTitleText,
    Tabs,
    Tab,
    Title,
    Divider,
    PageSection,
    Spinner,
    Bullseye,
} from '@patternfly/react-core';

import { fetchAlert } from 'services/AlertsService';
import PolicyDetailContent from 'Containers/Policies/Detail/PolicyDetailContent';
import { getClientWizardPolicy } from 'Containers/Policies/policies.utils';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';
import { Alert } from 'types/alert.proto';

import DeploymentDetails from './Deployment/DeploymentDetails';
import EnforcementDetails from './EnforcementDetails';
import ViolationNotFoundPage from '../ViolationNotFoundPage';
import ViolationDetails from './ViolationDetails';
import ViolationsBreadcrumbs from '../ViolationsBreadcrumbs';

function ViolationDetailsPage(): ReactElement {
    const isRouteEnabled = useIsRouteEnabled();
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForDeployment = hasReadAccess('Deployment');
    const isRouteEnabledForPolicy = isRouteEnabled('policy-management');

    const [activeTabKey, setActiveTabKey] = useState(0);
    const [alert, setAlert] = useState<Alert | null>(null);
    const [isFetchingSelectedAlert, setIsFetchingSelectedAlert] = useState(false);

    const { alertId } = useParams();

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
                    <Spinner isSVG />
                </Bullseye>
            );
        }
        return <ViolationNotFoundPage />;
    }

    const { policy, enforcement } = alert;
    const title = policy.name || 'Unknown violation';
    /* eslint-disable no-nested-ternary */
    const entityName =
        'resource' in alert
            ? alert.resource.clusterName
            : 'deployment' in alert
            ? alert.deployment.name
            : '';
    /* eslint-enable no-nested-ternary */
    const resourceType = 'resource' in alert ? alert.resource.resourceType : 'deployment';

    const displayedResourceType = startCase(resourceType.toLowerCase());

    return (
        <>
            <ViolationsBreadcrumbs current={title} />
            <PageSection variant="light">
                <Title headingLevel="h1">{title}</Title>
                <Title headingLevel="h2">{`in "${entityName}" ${displayedResourceType}`}</Title>
            </PageSection>
            <PageSection variant="default" padding={{ default: 'noPadding' }}>
                <Tabs
                    mountOnEnter
                    activeKey={activeTabKey}
                    onSelect={handleTabClick}
                    className="pf-u-background-color-100 pf-u-pl-lg"
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
                    {hasReadAccessForDeployment && 'deployment' in alert && (
                        <Tab eventKey={2} title={<TabTitleText>Deployment</TabTitleText>}>
                            <PageSection variant="default">
                                <DeploymentDetails alertDeployment={alert.deployment} />
                            </PageSection>
                        </Tab>
                    )}
                    {isRouteEnabledForPolicy && (
                        <Tab eventKey={3} title={<TabTitleText>Policy</TabTitleText>}>
                            <PageSection variant="default">
                                <>
                                    <Title headingLevel="h3" className="pf-u-mb-md">
                                        Policy overview
                                    </Title>
                                    <Divider component="div" className="pf-u-pb-md" />
                                    <PolicyDetailContent policy={getClientWizardPolicy(policy)} />
                                </>
                            </PageSection>
                        </Tab>
                    )}
                </Tabs>
            </PageSection>
        </>
    );
}

export default ViolationDetailsPage;
