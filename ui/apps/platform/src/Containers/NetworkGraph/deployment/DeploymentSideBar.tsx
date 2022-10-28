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
import DeploymentDetails from './DeploymentDetails';
import DeploymentNetworkPolicies from './DeploymentNetworkPolicies';

function DeploymentSideBar() {
    const { activeKeyTab, onSelectTab } = useTabs({
        defaultTab: 'Details',
    });

    return (
        <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }} className="pf-u-h-100">
            <Flex direction={{ default: 'row' }} className="pf-u-p-md pf-u-mb-0">
                <FlexItem>
                    <Badge style={{ backgroundColor: 'rgb(0,102,205)' }}>D</Badge>
                </FlexItem>
                <FlexItem>
                    <TextContent>
                        <Text component={TextVariants.h1} className="pf-u-font-size-xl">
                            visa-processor
                        </Text>
                    </TextContent>
                    <TextContent>
                        <Text
                            component={TextVariants.h2}
                            className="pf-u-font-size-sm pf-u-color-200"
                        >
                            in &quot;production / naples&quot;
                        </Text>
                    </TextContent>
                </FlexItem>
            </Flex>
            <FlexItem flex={{ default: 'flex_1' }}>
                <Tabs activeKey={activeKeyTab} onSelect={onSelectTab}>
                    <Tab
                        eventKey="Details"
                        tabContentId="Details"
                        title={<TabTitleText>Details</TabTitleText>}
                    />
                    <Tab
                        eventKey="Traffic"
                        tabContentId="Traffic"
                        title={<TabTitleText>Traffic</TabTitleText>}
                    />
                    <Tab
                        eventKey="Baselines"
                        tabContentId="Baselines"
                        title={<TabTitleText>Baselines</TabTitleText>}
                    />
                    <Tab
                        eventKey="Network policies"
                        tabContentId="Network policies"
                        title={<TabTitleText>Network policies</TabTitleText>}
                    />
                </Tabs>
                <TabContent eventKey="Details" id="Details" hidden={activeKeyTab !== 'Details'}>
                    <DeploymentDetails />
                </TabContent>
                <TabContent eventKey="Traffic" id="Traffic" hidden={activeKeyTab !== 'Traffic'}>
                    <div className="pf-u-h-100 pf-u-p-md">TODO: Add Traffic</div>
                </TabContent>
                <TabContent
                    eventKey="Baselines"
                    id="Baselines"
                    hidden={activeKeyTab !== 'Baselines'}
                >
                    <div className="pf-u-h-100 pf-u-p-md">TODO: Add Baselines</div>
                </TabContent>
                <TabContent
                    eventKey="Network policies"
                    id="Network policies"
                    hidden={activeKeyTab !== 'Network policies'}
                >
                    <DeploymentNetworkPolicies />
                </TabContent>
            </FlexItem>
        </Flex>
    );
}

export default DeploymentSideBar;
