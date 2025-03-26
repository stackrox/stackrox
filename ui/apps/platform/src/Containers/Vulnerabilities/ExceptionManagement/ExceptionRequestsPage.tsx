import React from 'react';
import {
    Flex,
    FlexItem,
    PageSection,
    Tab,
    TabTitleText,
    Tabs,
    Title,
} from '@patternfly/react-core';
import { Route, Routes, Navigate, useLocation, useNavigate } from 'react-router-dom';

import { exceptionManagementPath } from 'routePaths';

import PageTitle from 'Components/PageTitle';
import PendingRequests from './PendingRequests';
import ApprovedDeferrals from './ApprovedDeferrals';
import ApprovedFalsePositives from './ApprovedFalsePositives';
import DeniedRequests from './DeniedRequests';

type TabKey =
    | 'PENDING_REQUESTS'
    | 'APPROVED_DEFERRALS'
    | 'APPROVED_FALSE_POSITIVES'
    | 'DENIED_REQUESTS';

const pendingRequestsURL = `${exceptionManagementPath}/pending-requests`;
const approvedDeferralsURL = `${exceptionManagementPath}/approved-deferrals`;
const approvedFalsePositivesURL = `${exceptionManagementPath}/approved-false-positives`;
const deniedRequestsURL = `${exceptionManagementPath}/denied-requests`;

const tabKeyURLMap: Record<TabKey, string> = {
    PENDING_REQUESTS: pendingRequestsURL,
    APPROVED_DEFERRALS: approvedDeferralsURL,
    APPROVED_FALSE_POSITIVES: approvedFalsePositivesURL,
    DENIED_REQUESTS: deniedRequestsURL,
};

function ExceptionRequestsPage() {
    const location = useLocation();
    const navigate = useNavigate();

    let activeTabKey: TabKey = 'PENDING_REQUESTS';

    if (location.pathname === pendingRequestsURL) {
        activeTabKey = 'PENDING_REQUESTS';
    } else if (location.pathname === approvedDeferralsURL) {
        activeTabKey = 'APPROVED_DEFERRALS';
    } else if (location.pathname === approvedFalsePositivesURL) {
        activeTabKey = 'APPROVED_FALSE_POSITIVES';
    } else if (location.pathname === deniedRequestsURL) {
        activeTabKey = 'DENIED_REQUESTS';
    }

    const handleTabClick = (event, tabIndex) => {
        const path = tabKeyURLMap[tabIndex];
        const queryParams = location.search;
        // If you're manipulating the query parameters before navigating, consider improving this URL construction
        const url = `${path}${queryParams}`;
        navigate(url);
    };

    /* eslint-disable accessibility/Tab-empty-contentId */
    // ROX-25890 after React Router upgrade:
    // Add tabContentId prop to Tab elements.
    // Add TabContent elements with id props in routes.
    return (
        <>
            <PageTitle title="Exception Management" />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-row pf-v5-u-align-items-center"
                variant="light"
            >
                <Flex direction={{ default: 'column' }}>
                    <Title headingLevel="h1">Exception management</Title>
                    <FlexItem>
                        Approve or deny requests for vulnerability deferrals and false positives.
                    </FlexItem>
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <Tabs
                    activeKey={activeTabKey}
                    onSelect={handleTabClick}
                    className="pf-v5-u-pl-lg pf-v5-u-background-color-100"
                >
                    <Tab
                        eventKey="PENDING_REQUESTS"
                        title={<TabTitleText>Pending requests</TabTitleText>}
                    />
                    <Tab
                        eventKey="APPROVED_DEFERRALS"
                        title={<TabTitleText>Approved deferrals</TabTitleText>}
                    />
                    <Tab
                        eventKey="APPROVED_FALSE_POSITIVES"
                        title={<TabTitleText>Approved false positives</TabTitleText>}
                    />
                    <Tab
                        eventKey="DENIED_REQUESTS"
                        title={<TabTitleText>Denied requests</TabTitleText>}
                    />
                </Tabs>
                <Routes>
                    <Route index element={<Navigate to={pendingRequestsURL} replace />} />
                    <Route path="pending-requests" element={<PendingRequests />} />
                    <Route path="approved-deferrals" element={<ApprovedDeferrals />} />
                    <Route path="approved-false-positives" element={<ApprovedFalsePositives />} />
                    <Route path="denied-requests" element={<DeniedRequests />} />
                </Routes>
            </PageSection>
        </>
    );
}

export default ExceptionRequestsPage;
