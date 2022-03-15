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
import { preFormatPolicyFields } from 'Containers/Policies/Wizard/Form/utils';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import PolicyDetailContent from '../../Policies/PatternFly/Detail/PolicyDetailContent';
import DeploymentDetails from './DeploymentDetails';
import PolicyDetails from './PolicyDetails';
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
    const isPoliciesPFEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_POLICIES_PATTERNFLY);

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

    const { policy, deployment, resource, commonEntityInfo } = alert;
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
            <PageSection
                variant="default"
                padding={{ default: 'noPadding' }}
                data-testid="violation-details-page"
            >
                <Tabs
                    mountOnEnter
                    activeKey={activeTabKey}
                    onSelect={handleTabClick}
                    className="pf-u-background-color-100 pf-u-pl-lg"
                >
                    <Tab eventKey={0} title={<TabTitleText>Violation</TabTitleText>}>
                        <PageSection variant="default">
                            <ViolationDetails
                                violationId={alert.id}
                                violations={alert.violations}
                                processViolation={alert.processViolation}
                                lifecycleStage={alert.lifecycleStage}
                            />
                        </PageSection>
                    </Tab>
                    {alert?.enforcement && (
                        <Tab eventKey={1} title={<TabTitleText>Enforcement</TabTitleText>}>
                            <PageSection variant="default">
                                <EnforcementDetails alert={alert} />
                            </PageSection>
                        </Tab>
                    )}
                    {alert?.deployment && (
                        <Tab eventKey={2} title={<TabTitleText>Deployment</TabTitleText>}>
                            <PageSection variant="default">
                                <DeploymentDetails deployment={alert.deployment} />
                            </PageSection>
                        </Tab>
                    )}
                    <Tab eventKey={3} title={<TabTitleText>Policy</TabTitleText>}>
                        <PageSection variant="default">
                            {isPoliciesPFEnabled ? (
                                <>
                                    <Title headingLevel="h3" className="pf-u-mb-md">
                                        Policy overview
                                    </Title>
                                    <Divider component="div" className="pf-u-pb-md" />
                                    <PolicyDetailContent policy={policy} />
                                </>
                            ) : (
                                <PolicyDetails policy={preFormatPolicyFields(alert.policy)} />
                            )}
                        </PageSection>
                    </Tab>
                </Tabs>
            </PageSection>
        </>
    );
}

export default ViolationDetailsPage;
