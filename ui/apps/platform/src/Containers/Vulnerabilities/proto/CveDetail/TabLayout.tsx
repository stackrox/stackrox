import { useState } from 'react';
import { Stack, StackItem, Tab, Tabs, Title } from '@patternfly/react-core';

import type { ProtoAdvisory, ProtoComponent, ProtoImage } from './useCveDetail';
import AdvisoriesTable from './AdvisoriesTable';
import AffectedComponentsTable from './AffectedComponentsTable';
import AffectedImagesTable from './AffectedImagesTable';

type TabLayoutProps = {
    advisories: ProtoAdvisory[];
    components: ProtoComponent[];
    images: ProtoImage[];
};

/**
 * Advisories always visible at the top; Components and Images in tabs below.
 */
function TabLayout({ advisories, components, images }: TabLayoutProps) {
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
                        <AffectedComponentsTable components={components} />
                    </Tab>
                    <Tab eventKey="images" title="Affected Images">
                        <AffectedImagesTable images={images} />
                    </Tab>
                </Tabs>
            </StackItem>
        </Stack>
    );
}

export default TabLayout;
