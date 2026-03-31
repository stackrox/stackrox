import type { MouseEvent } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import { PageSection, Tab, TabTitleText, Tabs, Title } from '@patternfly/react-core';

import { policiesBasePath, policyCategoriesPath } from 'routePaths';

type PolicyManagementHeaderProps = {
    currentTabTitle: string;
};

function PolicyManagementHeader({ currentTabTitle }: PolicyManagementHeaderProps) {
    const navigate = useNavigate();

    const handleTabSelect = (_event: MouseEvent<HTMLElement>, tabKey: string | number) => {
        if (tabKey === 'Policies') {
            navigate(policiesBasePath);
        } else if (tabKey === 'Policy categories') {
            navigate(policyCategoriesPath);
        }
    };

    return (
        <>
            <PageSection>
                <Title headingLevel="h1">Policy management</Title>
            </PageSection>
            <PageSection type="tabs">
                <Tabs activeKey={currentTabTitle} onSelect={handleTabSelect} usePageInsets>
                    <Tab
                        eventKey="Policies"
                        title={<TabTitleText>Policies</TabTitleText>}
                        tabContentId="policies-table"
                    />
                    <Tab
                        eventKey="Policy categories"
                        title={<TabTitleText>Policy categories</TabTitleText>}
                        tabContentId="policy-categories-list-section"
                    />
                </Tabs>
            </PageSection>
        </>
    );
}

export default PolicyManagementHeader;
