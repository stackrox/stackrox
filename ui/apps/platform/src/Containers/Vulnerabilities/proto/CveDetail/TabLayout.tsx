import { useState } from 'react';
import { Stack, StackItem, Tab, Tabs, Title } from '@patternfly/react-core';

import type { ProtoAdvisory } from './useCveDetail';
import AdvisoriesTable from './AdvisoriesTable';
import AffectedComponentsTable from './AffectedComponentsTable';
import AffectedImagesTable from './AffectedImagesTable';

type TabLayoutProps = {
    advisories: ProtoAdvisory[];
};

/**
 * Advisories always visible at the top; Components and Images in tabs below.
 */
function TabLayout({ advisories }: TabLayoutProps) {
    const [activeTab, setActiveTab] = useState<string | number>('components');

    return (
        <Stack hasGutter>
            <StackItem>
                <Title headingLevel="h3">Advisories</Title>
                <AdvisoriesTable advisories={advisories} />
            </StackItem>
            <StackItem>
                <Tabs
                    activeKey={activeTab}
                    onSelect={(_event, tabKey) => setActiveTab(tabKey)}
                    aria-label="Detail sections"
                >
                    <Tab eventKey="components" title="Affected Components">
                        <AffectedComponentsTable />
                    </Tab>
                    <Tab eventKey="images" title="Affected Images">
                        <AffectedImagesTable />
                    </Tab>
                </Tabs>
            </StackItem>
        </Stack>
    );
}

export default TabLayout;
