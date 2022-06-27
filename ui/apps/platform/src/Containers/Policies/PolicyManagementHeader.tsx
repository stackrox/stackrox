import React from 'react';
import { PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import TabNav from 'Components/TabNav';
import { policiesBasePath, policyCategoriesPath } from 'routePaths';

type PolicyManagementHeaderProps = {
    currentTabTitle?: string;
};

function PolicyManagementHeader({ currentTabTitle }: PolicyManagementHeaderProps) {
    return (
        <>
            <PageTitle title="Policy management - Policy categories" />
            <PageSection variant="light">
                <Title headingLevel="h1">Policy management</Title>
            </PageSection>
            <PageSection variant="light" className="pf-u-px-sm pf-u-py-0">
                <TabNav
                    currentTabTitle={currentTabTitle}
                    tabLinks={[
                        { title: 'Policies', href: policiesBasePath },
                        { title: 'Policy categories', href: policyCategoriesPath },
                    ]}
                />
            </PageSection>
        </>
    );
}

export default PolicyManagementHeader;
