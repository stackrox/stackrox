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

function DeploymentSideBar() {
    const { activeKeyTab, onSelectTab } = useTabs({
        defaultTab: 'Details',
    });

    return (
        <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }} className="pf-u-h-100">
            <Flex direction={{ default: 'row' }} className="pf-u-p-md">
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
                        eventKey="Flows"
                        tabContentId="Flows"
                        title={<TabTitleText>Flows</TabTitleText>}
                    />
                    <Tab
                        eventKey="Baseline"
                        tabContentId="Baseline"
                        title={<TabTitleText>Baseline</TabTitleText>}
                    />
                    <Tab
                        eventKey="Policies"
                        tabContentId="Policies"
                        title={<TabTitleText>Policies</TabTitleText>}
                    />
                    <Tab
                        eventKey="Timeline"
                        tabContentId="Timeline"
                        title={<TabTitleText>Timeline</TabTitleText>}
                    />
                </Tabs>
                <TabContent eventKey="Details" id="Details" hidden={activeKeyTab !== 'Details'}>
                    <DeploymentDetails />
                </TabContent>
                <TabContent eventKey="Flows" id="Flows" hidden={activeKeyTab !== 'Flows'}>
                    <div className="pf-u-h-100 pf-u-p-md">TODO: Add Flows</div>
                </TabContent>
                <TabContent eventKey="Baseline" id="Baseline" hidden={activeKeyTab !== 'Baseline'}>
                    <div className="pf-u-h-100 pf-u-p-md">TODO: Add Baseline</div>
                </TabContent>
                <TabContent eventKey="Policies" id="Policies" hidden={activeKeyTab !== 'Policies'}>
                    <div className="pf-u-h-100 pf-u-p-md">TODO: Add Policies</div>
                </TabContent>
                <TabContent eventKey="Timeline" id="Timeline" hidden={activeKeyTab !== 'Timeline'}>
                    <div className="pf-u-h-100 pf-u-p-md">TODO: Add Timeline</div>
                </TabContent>
            </FlexItem>
        </Flex>
    );
}

export default DeploymentSideBar;
