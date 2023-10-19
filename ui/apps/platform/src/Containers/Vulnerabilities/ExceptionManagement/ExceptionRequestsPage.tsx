import React, { useState } from 'react';
import {
    Flex,
    FlexItem,
    PageSection,
    Tab,
    TabTitleText,
    Tabs,
    Title,
} from '@patternfly/react-core';

type TabKey =
    | 'PENDING_REQUESTS'
    | 'APPROVED_DEFERRALS'
    | 'APPROVED_FALSE_POSITIVES'
    | 'DENIED_REQUESTS';

function ExceptionRequestsPage() {
    const [activeTabKey, setActiveTabKey] = useState<TabKey>('PENDING_REQUESTS');

    const handleTabClick = (event, tabIndex) => {
        setActiveTabKey(tabIndex);
    };

    return (
        <>
            <PageSection
                className="pf-u-display-flex pf-u-flex-direction-row pf-u-align-items-center"
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
                    component="nav"
                    className="pf-u-pl-lg pf-u-background-color-100"
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
            </PageSection>
        </>
    );
}

export default ExceptionRequestsPage;
