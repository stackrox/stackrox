import React from 'react';
import {
    Badge,
    Flex,
    FlexItem,
    Tab,
    TabContent,
    Tabs,
    TabTitleText,
    Text,
    TextContent,
    TextVariants,
} from '@patternfly/react-core';

import useTabs from 'hooks/patternfly/useTabs';
import NamespaceDeployments from './NamespaceDeployments';
import NamespaceNetworkPolicies from './NamespaceNetworkPolicies';

function NamespaceSideBar() {
    const { activeKeyTab, onSelectTab } = useTabs({
        defaultTab: 'Deployments',
    });

    return (
        <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }} className="pf-u-h-100">
            <Flex direction={{ default: 'row' }} className="pf-u-p-md pf-u-mb-0">
                <FlexItem>
                    <Badge style={{ backgroundColor: 'rgb(32,79,23)' }}>NS</Badge>
                </FlexItem>
                <FlexItem>
                    <TextContent>
                        <Text component={TextVariants.h1} className="pf-u-font-size-xl">
                            stackrox
                        </Text>
                    </TextContent>
                    <TextContent>
                        <Text
                            component={TextVariants.h2}
                            className="pf-u-font-size-sm pf-u-color-200"
                        >
                            in &quot;remote&quot;
                        </Text>
                    </TextContent>
                </FlexItem>
            </Flex>
            <FlexItem flex={{ default: 'flex_1' }}>
                <Tabs activeKey={activeKeyTab} onSelect={onSelectTab}>
                    <Tab
                        eventKey="Deployments"
                        tabContentId="Deployments"
                        title={<TabTitleText>Deployments</TabTitleText>}
                    />
                    <Tab
                        eventKey="Network policies"
                        tabContentId="Network policies"
                        title={<TabTitleText>Network policies</TabTitleText>}
                    />
                </Tabs>
                <TabContent
                    eventKey="Deployments"
                    id="Deployments"
                    hidden={activeKeyTab !== 'Deployments'}
                >
                    <NamespaceDeployments />
                </TabContent>
                <TabContent
                    eventKey="Network policies"
                    id="Network policies"
                    hidden={activeKeyTab !== 'Network policies'}
                >
                    <NamespaceNetworkPolicies />
                </TabContent>
            </FlexItem>
        </Flex>
    );
}

export default NamespaceSideBar;
