import { PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import TabNav from 'Components/TabNav/TabNav';

type TabNavHeaderProps = {
    mainTitle: string;
    pageTitle: string;
    currentTabTitle: string;
    tabLinks: { title: string; href: string }[];
};

function TabNavHeader({ mainTitle, pageTitle, currentTabTitle, tabLinks }: TabNavHeaderProps) {
    return (
        <>
            <PageTitle title={pageTitle} />
            <PageSection variant="light">
                <Title headingLevel="h1">{mainTitle}</Title>
            </PageSection>
            <PageSection variant="light" className="pf-v5-u-px-sm pf-v5-u-py-0">
                <TabNav currentTabTitle={currentTabTitle} tabLinks={tabLinks} />
            </PageSection>
        </>
    );
}

export default TabNavHeader;
