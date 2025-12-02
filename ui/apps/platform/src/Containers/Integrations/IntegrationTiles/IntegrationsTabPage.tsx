import type { ReactElement, ReactNode } from 'react';
import { Divider, Flex, FlexItem, PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import TabNav from 'Components/TabNav/TabNav';
import type { IntegrationSource } from 'types/integration';

import OcmDeprecatedToken from '../Banners/OcmDeprecatedToken';
import { getIntegrationTabPath, integrationSourceTitleMap } from '../utils/integrationsList';

export type IntegrationsTabPageProps = {
    children: ReactNode;
    source: IntegrationSource;
    sourcesEnabled: IntegrationSource[];
};

function IntegrationsTabPage({
    children,
    source,
    sourcesEnabled,
}: IntegrationsTabPageProps): ReactElement {
    const sourceTitle = integrationSourceTitleMap[source];
    const tabLinks = sourcesEnabled.map((sourceEnabled) => ({
        href: getIntegrationTabPath(sourceEnabled),
        title: integrationSourceTitleMap[sourceEnabled],
    }));

    return (
        <>
            <PageTitle title={`Integrations - ${sourceTitle}`} />
            <PageSection hasBodyWrapper={false} component="div">
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                    <Title headingLevel="h1">Integrations</Title>
                    {/*TODO(ROX-25633): Remove the banner again.*/}
                    <OcmDeprecatedToken />
                    <FlexItem>
                        <TabNav tabLinks={tabLinks} currentTabTitle={sourceTitle} />
                        <Divider component="div" />
                    </FlexItem>
                    {children}
                </Flex>
            </PageSection>
        </>
    );
}

export default IntegrationsTabPage;
