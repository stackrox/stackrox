import React, { ReactElement } from 'react';
import {
    Divider,
    Flex,
    FlexItem,
    PageSection,
    PageSectionVariants,
    Tab,
    TabContent,
    Tabs,
    TabTitleText,
    TextContent,
    Title,
} from '@patternfly/react-core';
import { Switch, Route, useHistory, useLocation, Redirect } from 'react-router-dom';

import {
    dashboardPath,
    vulnManagementApprovedDeferralsPath,
    vulnManagementApprovedFalsePositivesPath,
    vulnManagementPath,
    vulnManagementPendingApprovalsPath,
} from 'routePaths';
import usePermissions from 'hooks/usePermissions';
import NotFoundMessage from 'Components/NotFoundMessage';
import PendingApprovals from './PendingApprovals';
import ApprovedDeferrals from './ApprovedDeferrals';
import ApprovedFalsePositives from './ApprovedFalsePositives';

const TABS = {
    PENDING_APPROVALS: 'pending-approvals-tab',
    APPROVED_DEFERRALS: 'approved-deferrals-tab',
    APPROVED_FALSE_POSITIVES: 'approved-false-positives-tab',
};

const TAB_LABELS = {
    PENDING_APPROVALS: 'Pending Approvals',
    APPROVED_DEFERRALS: 'Approved Deferrals',
    APPROVED_FALSE_POSITIVES: 'Approved False Positives',
};

function getActiveKeyTab(pathname: string) {
    if (pathname === vulnManagementPendingApprovalsPath) {
        return TABS.PENDING_APPROVALS;
    }
    if (pathname === vulnManagementApprovedDeferralsPath) {
        return TABS.APPROVED_DEFERRALS;
    }
    if (pathname === vulnManagementApprovedFalsePositivesPath) {
        return TABS.APPROVED_FALSE_POSITIVES;
    }
    return null;
}

function TabContentList({ activeKeyTab }): ReactElement {
    return (
        <>
            <TabContent
                eventKey={TABS.PENDING_APPROVALS}
                id={TABS.PENDING_APPROVALS}
                hidden={activeKeyTab !== TABS.PENDING_APPROVALS}
            >
                <PendingApprovals />
            </TabContent>
            <TabContent
                eventKey={TABS.APPROVED_DEFERRALS}
                id={TABS.APPROVED_DEFERRALS}
                hidden={activeKeyTab !== TABS.APPROVED_DEFERRALS}
            >
                <ApprovedDeferrals />
            </TabContent>
            <TabContent
                eventKey={TABS.APPROVED_FALSE_POSITIVES}
                id={TABS.APPROVED_FALSE_POSITIVES}
                hidden={activeKeyTab !== TABS.APPROVED_FALSE_POSITIVES}
            >
                <ApprovedFalsePositives />
            </TabContent>
        </>
    );
}

function RiskAcceptancePage(): ReactElement {
    const { hasReadAccess } = usePermissions();
    const history = useHistory();
    const location = useLocation();

    if (
        !hasReadAccess('VulnerabilityManagementRequests') ||
        !hasReadAccess('VulnerabilityManagementApprovals')
    ) {
        return (
            <NotFoundMessage
                title="404: Not found"
                message="This page doesn't exist, return to Vulnerability Management"
                actionText="Go to Vulnerability Management"
                url={vulnManagementPath}
            />
        );
    }

    const activeKeyTab = getActiveKeyTab(location.pathname);

    if (!activeKeyTab) {
        return <Redirect to={vulnManagementPendingApprovalsPath} />;
    }

    function onSelectTab(_, eventKey) {
        if (eventKey === TABS.PENDING_APPROVALS) {
            history.push(vulnManagementPendingApprovalsPath);
        } else if (eventKey === TABS.APPROVED_DEFERRALS) {
            history.push(vulnManagementApprovedDeferralsPath);
        } else if (eventKey === TABS.APPROVED_FALSE_POSITIVES) {
            history.push(vulnManagementApprovedFalsePositivesPath);
        } else {
            history.push(dashboardPath);
        }
    }

    return (
        <>
            <PageSection variant={PageSectionVariants.light}>
                <Flex
                    alignItems={{
                        default: 'alignItemsFlexStart',
                        md: 'alignItemsCenter',
                    }}
                    direction={{ default: 'column', md: 'row' }}
                    flexWrap={{ default: 'nowrap' }}
                    spaceItems={{ default: 'spaceItemsXl' }}
                >
                    <FlexItem grow={{ default: 'grow' }}>
                        <TextContent>
                            <Title headingLevel="h1">Risk Acceptance</Title>
                        </TextContent>
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <div>
                <Tabs activeKey={activeKeyTab} onSelect={onSelectTab}>
                    <Tab
                        eventKey={TABS.PENDING_APPROVALS}
                        tabContentId={TABS.PENDING_APPROVALS}
                        title={<TabTitleText>{TAB_LABELS.PENDING_APPROVALS}</TabTitleText>}
                    />
                    <Tab
                        eventKey={TABS.APPROVED_DEFERRALS}
                        tabContentId={TABS.APPROVED_DEFERRALS}
                        title={<TabTitleText>{TAB_LABELS.APPROVED_DEFERRALS}</TabTitleText>}
                    />
                    <Tab
                        eventKey={TABS.APPROVED_FALSE_POSITIVES}
                        tabContentId={TABS.APPROVED_FALSE_POSITIVES}
                        title={<TabTitleText>{TAB_LABELS.APPROVED_FALSE_POSITIVES}</TabTitleText>}
                    />
                </Tabs>
            </div>
            <Switch>
                <Route
                    exact
                    path={vulnManagementPendingApprovalsPath}
                    render={() => <TabContentList activeKeyTab={activeKeyTab} />}
                />
                <Route
                    exact
                    path={vulnManagementApprovedDeferralsPath}
                    render={() => <TabContentList activeKeyTab={activeKeyTab} />}
                />
                <Route
                    exact
                    path={vulnManagementApprovedFalsePositivesPath}
                    render={() => <TabContentList activeKeyTab={activeKeyTab} />}
                />
            </Switch>
        </>
    );
}

export default RiskAcceptancePage;
