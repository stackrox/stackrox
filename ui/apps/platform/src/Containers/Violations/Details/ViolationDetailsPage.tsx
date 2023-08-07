import React, { ReactElement, useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
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
import DeploymentDetails from './Deployment/DeploymentDetails';
import EnforcementDetails from './EnforcementDetails';
import { Alert } from '../types/violationTypes';
import ViolationNotFoundPage from '../ViolationNotFoundPage';
import ViolationDetails from './ViolationDetails';
import ViolationsBreadcrumbs from '../ViolationsBreadcrumbs';

function ViolationDetailsPage(): ReactElement {
    const [activeTabKey, setActiveTabKey] = useState(0);
    const [alert, setAlert] = useState<Alert>();
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
                setAlert(undefined);
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

    const { policy, deployment, resource, commonEntityInfo, enforcement } = alert;
    const title = policy.name || 'Unknown violation';
    const { name: entityName } = resource || deployment || {};
    const resourceType = resource?.resourceType || commonEntityInfo?.resourceType || 'deployment';

    return (
        <>
            <ViolationsBreadcrumbs current={title} />
            <PageSection variant="light">
                <Title headingLevel="h1">{title}</Title>
                <Title headingLevel="h2">{`in "${entityName as string}" ${resourceType}`}</Title>
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
                    {deployment && (
                        <Tab eventKey={2} title={<TabTitleText>Deployment</TabTitleText>}>
                            <PageSection variant="default">
                                <DeploymentDetails alertDeployment={deployment} />
                            </PageSection>
                        </Tab>
                    )}
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
                </Tabs>
            </PageSection>
        </>
    );
}

export default ViolationDetailsPage;
