import React, { ReactElement, useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import {
    Breadcrumb,
    BreadcrumbItem,
    TabTitleText,
    Tabs,
    Tab,
    Title,
    Divider,
    PageSection,
    Spinner,
    Bullseye,
} from '@patternfly/react-core';

import { violationsBasePath } from 'routePaths';
import { fetchAlert } from 'services/AlertsService';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { preFormatPolicyFields } from 'Containers/Policies/Wizard/Form/utils';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import PolicyDetailContent from '../../../Policies/PatternFly/Detail/PolicyDetailContent';
import DeploymentDetails from './DeploymentDetails';
import PolicyDetails from './PolicyDetails';
import EnforcementDetails from './EnforcementDetails';
import { Alert } from '../types/violationTypes';
import ViolationNotFoundPage from '../ViolationNotFoundPage';
import ViolationDetails from './ViolationDetails';

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
        <PageSection variant="light" isFilled data-testid="violation-details-page">
            <Breadcrumb className="pf-u-mb-md">
                <BreadcrumbItemLink to={violationsBasePath}>Violations</BreadcrumbItemLink>
                <BreadcrumbItem isActive>{title}</BreadcrumbItem>
            </Breadcrumb>
            <Title headingLevel="h1">{title}</Title>
            <Title headingLevel="h2" className="pf-u-mb-md">{`in "${
                entityName as string
            }" ${resourceType}`}</Title>
            <Tabs mountOnEnter activeKey={activeTabKey} onSelect={handleTabClick}>
                <Tab eventKey={0} title={<TabTitleText>Violation</TabTitleText>}>
                    <ViolationDetails
                        violationId={alert.id}
                        violations={alert.violations}
                        processViolation={alert.processViolation}
                        lifecycleStage={alert.lifecycleStage}
                    />
                </Tab>
                {alert?.enforcement && (
                    <Tab eventKey={1} title={<TabTitleText>Enforcement</TabTitleText>}>
                        <EnforcementDetails alert={alert} />
                    </Tab>
                )}
                {alert?.deployment && (
                    <Tab eventKey={2} title={<TabTitleText>Deployment</TabTitleText>}>
                        <DeploymentDetails deployment={alert.deployment} />
                    </Tab>
                )}
                <Tab eventKey={3} title={<TabTitleText>Policy</TabTitleText>}>
                    {isPoliciesPFEnabled ? (
                        <>
                            <Title headingLevel="h2" className="pf-u-my-md">
                                Policy overview
                            </Title>
                            <Divider component="div" className="pf-u-pb-md" />
                            <PolicyDetailContent policy={policy} notifiers={[]} clusters={[]} />
                        </>
                    ) : (
                        <PolicyDetails policy={preFormatPolicyFields(alert.policy)} />
                    )}
                </Tab>
            </Tabs>
        </PageSection>
    );
}

export default ViolationDetailsPage;
