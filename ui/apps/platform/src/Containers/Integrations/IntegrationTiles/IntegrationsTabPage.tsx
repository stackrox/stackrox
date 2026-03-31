import type { ReactElement, ReactNode } from 'react';
import { Flex, PageSection, Tab, TabTitleText, Tabs, Title } from '@patternfly/react-core';
import { useNavigate } from 'react-router-dom-v5-compat';

import PageTitle from 'Components/PageTitle';
import { isIntegrationSource } from 'types/integration';
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
    const navigate = useNavigate();

    return (
        <>
            <PageTitle title={`Integrations - ${sourceTitle}`} />
            <PageSection>
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                    <Title headingLevel="h1">Integrations</Title>
                    {/*TODO(ROX-25633): Remove the banner again.*/}
                    <OcmDeprecatedToken />
                </Flex>
            </PageSection>
            <PageSection type="tabs">
                <Tabs
                    activeKey={source}
                    onSelect={(_event, tabKey) => {
                        if (isIntegrationSource(tabKey)) {
                            navigate(getIntegrationTabPath(tabKey));
                        }
                    }}
                    usePageInsets
                    mountOnEnter
                    unmountOnExit
                >
                    {sourcesEnabled.map((sourceEnabled) => (
                        <Tab
                            key={sourceEnabled}
                            eventKey={sourceEnabled}
                            title={
                                <TabTitleText>
                                    {integrationSourceTitleMap[sourceEnabled]}
                                </TabTitleText>
                            }
                            tabContentId={sourceEnabled}
                        >
                            <PageSection>{children}</PageSection>
                        </Tab>
                    ))}
                </Tabs>
            </PageSection>
        </>
    );
}

export default IntegrationsTabPage;
