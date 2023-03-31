import React from 'react';
import { gql, useQuery } from '@apollo/client';
import {
    Tabs,
    Tab,
    TabTitleText,
    TabsComponent,
    PageSection,
    Card,
    CardBody,
    Divider,
    Toolbar,
    ToolbarContent,
} from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import WorkloadTableToolbar from './WorkloadTableToolbar';
import EntityTypeToggleGroup, {
    imageCountQuery,
    deploymentCountQuery,
    cveCountQuery,
} from './EntityTypeToggleGroup';
import { DefaultFilters, cveStatusTabValues } from './types';
import TablePaginationControls from './TablePaginationControls';
import WorkloadCvesTable from './WorkloadCvesTable';

type CveStatusTabNavigationProps = {
    defaultFilters: DefaultFilters;
};

function CveStatusTabNavigation({ defaultFilters }: CveStatusTabNavigationProps) {
    const [activeCVEStatusKey, setActiveCVEStatusKey] = useURLStringUnion(
        'cveStatus',
        cveStatusTabValues
    );

    function handleTabClick(e, tab) {
        setActiveCVEStatusKey(tab);
    }

    return (
        <Tabs
            activeKey={activeCVEStatusKey}
            onSelect={handleTabClick}
            component={TabsComponent.nav}
            className="pf-u-pl-lg pf-u-background-color-100"
            mountOnEnter
            unmountOnExit
        >
            <Tab eventKey="Observed" title={<TabTitleText>Observed CVEs</TabTitleText>}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            <WorkloadTableToolbar defaultFilters={defaultFilters} />
                            <Divider component="div" />
                            <Toolbar>
                                <ToolbarContent>
                                    <EntityTypeToggleGroup />
                                    <Divider orientation={{ default: 'vertical' }} />
                                    <TablePaginationControls />
                                </ToolbarContent>
                            </Toolbar>
                            <Divider component="div" />
                            <WorkloadCvesTable entity="Image" />
                        </CardBody>
                    </Card>
                </PageSection>
            </Tab>
            <Tab eventKey="Deferred" title={<TabTitleText>Deferrals</TabTitleText>} isDisabled>
                deferrals tbd
            </Tab>
            <Tab
                eventKey="False Positive"
                title={<TabTitleText>False Positives</TabTitleText>}
                isDisabled
            >
                False-positives tbd
            </Tab>
        </Tabs>
    );
}

export default CveStatusTabNavigation;
