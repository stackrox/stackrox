import React, { useState } from 'react';
import {
    Tabs,
    Tab,
    TabTitleText,
    TabsComponent,
    PageSection,
    Card,
    CardBody,
} from '@patternfly/react-core';

import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import { getQueryString } from 'utils/queryStringUtils';
import WorkloadTableToolbar from './WorkloadTableToolbar';
import EntityTypeToggleGroup from './EntityTypeToggleGroup';
import { WorkloadCvesSearch } from './searchUtils';
import { DefaultFilters } from './types';

const observedCvesQueryString = getQueryString<WorkloadCvesSearch>({ cveStatusTab: 'Observed' });
const observedCvesPath = `${vulnerabilitiesWorkloadCvesPath}${observedCvesQueryString}`;

type CveStatusTabNavigationProps = {
    defaultFilters: DefaultFilters;
};

function CveStatusTabNavigation({ defaultFilters }: CveStatusTabNavigationProps) {
    const [activeTabKey, setActiveTabKey] = useState(0);

    function handleTabClick(e, tabIndex) {
        setActiveTabKey(tabIndex);
    }

    return (
        <Tabs
            activeKey={activeTabKey}
            onSelect={handleTabClick}
            component={TabsComponent.nav}
            className="pf-u-pl-lg pf-u-background-color-100"
            mountOnEnter
            unmountOnExit
        >
            <Tab
                eventKey={0}
                title={<TabTitleText>Observed CVEs</TabTitleText>}
                href={observedCvesPath}
            >
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            <WorkloadTableToolbar defaultFilters={defaultFilters} />
                            <EntityTypeToggleGroup />
                            cve overview table here
                        </CardBody>
                    </Card>
                </PageSection>
            </Tab>
            <Tab eventKey={1} title={<TabTitleText>Deferrals</TabTitleText>} isDisabled>
                deferrals tbd
            </Tab>
            <Tab eventKey={2} title={<TabTitleText>False Positives</TabTitleText>} isDisabled>
                False-positives tbd
            </Tab>
        </Tabs>
    );
}

export default CveStatusTabNavigation;
