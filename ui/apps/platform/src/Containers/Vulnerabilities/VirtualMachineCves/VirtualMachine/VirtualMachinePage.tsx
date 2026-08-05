import { useCallback } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import {
    Breadcrumb,
    BreadcrumbItem,
    Content,
    Divider,
    PageSection,
    Skeleton,
    Tab,
    TabContent,
    Tabs,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useRestQuery from 'hooks/useRestQuery';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { getVM } from 'services/VirtualMachineService';

import { detailsTabValues } from '../../types';
import { getOverviewPagePath } from '../../utils/searchUtils';
import VirtualMachinePageHeader from './VirtualMachinePageHeader';
import VirtualMachinePageComponents from './VirtualMachinePageComponents';
import VirtualMachinePageDetails from './VirtualMachinePageDetails';
import VirtualMachinePageVulnerabilities from './VirtualMachinePageVulnerabilities';

const VULNERABILITIES_TAB_ID = 'vulnerabilities-tab-content';
const COMPONENTS_TAB_ID = 'components-tab-content';
const DETAILS_TAB_ID = 'details-tab-content';

const virtualMachineCveOverviewPath = getOverviewPagePath('VirtualMachine', {
    entityTab: 'VirtualMachine',
});

function VirtualMachinePage() {
    const { virtualMachineId } = useParams() as { virtualMachineId: string };

    const fetchVirtualMachine = useCallback(() => getVM(virtualMachineId), [virtualMachineId]);
    const { data: virtualMachineDetail, isLoading, error } = useRestQuery(fetchVirtualMachine);

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);

    const vulnTabKey = detailsTabValues[0];
    const componentsTabKey = detailsTabValues[4];
    const detailsTabKey = detailsTabValues[1];

    function onTabChange(value: string | number) {
        setActiveTabKey(value);
    }

    return (
        <>
            <PageTitle
                title={`Virtual Machine CVEs - Virtual Machine ${virtualMachineDetail?.name}`}
            />
            <PageSection>
                <Breadcrumb>
                    <BreadcrumbItemLink to={virtualMachineCveOverviewPath}>
                        Virtual Machines
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {virtualMachineDetail?.name ?? (
                            <Skeleton
                                screenreaderText="Loading Virtual Machine name"
                                width="200px"
                            />
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <VirtualMachinePageHeader
                    virtualMachineDetail={virtualMachineDetail}
                    isLoading={isLoading}
                    error={error}
                />
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <Tabs
                    activeKey={activeTabKey}
                    onSelect={(_, key) => {
                        onTabChange(key);
                    }}
                    className="pf-v6-u-pl-md"
                >
                    <Tab
                        eventKey={vulnTabKey}
                        tabContentId={VULNERABILITIES_TAB_ID}
                        title={vulnTabKey}
                    />
                    <Tab
                        eventKey={componentsTabKey}
                        tabContentId={COMPONENTS_TAB_ID}
                        title={componentsTabKey}
                    />
                    <Tab
                        eventKey={detailsTabKey}
                        tabContentId={DETAILS_TAB_ID}
                        title={detailsTabKey}
                    />
                </Tabs>
            </PageSection>
            <PageSection padding={{ default: 'padding' }}>
                <Content component="p">
                    <Content component="p">
                        {activeTabKey === vulnTabKey &&
                            'Prioritize and remediate observed CVEs for this virtual machine'}
                        {activeTabKey === componentsTabKey &&
                            'View all components from this virtual machine'}
                        {activeTabKey === detailsTabKey &&
                            'View details about this virtual machine'}
                    </Content>
                </Content>
            </PageSection>
            <PageSection
                isFilled
                padding={{ default: 'padding' }}
                aria-label={activeTabKey}
                role="tabpanel"
                tabIndex={0}
            >
                {activeTabKey === vulnTabKey && (
                    <TabContent id={VULNERABILITIES_TAB_ID}>
                        <VirtualMachinePageVulnerabilities virtualMachineId={virtualMachineId} />
                    </TabContent>
                )}
                {activeTabKey === componentsTabKey && (
                    <TabContent id={COMPONENTS_TAB_ID}>
                        <VirtualMachinePageComponents virtualMachineId={virtualMachineId} />
                    </TabContent>
                )}
                {activeTabKey === detailsTabKey && (
                    <TabContent id={DETAILS_TAB_ID}>
                        {virtualMachineDetail && (
                            <VirtualMachinePageDetails
                                virtualMachineDetail={virtualMachineDetail}
                            />
                        )}
                    </TabContent>
                )}
            </PageSection>
        </>
    );
}

export default VirtualMachinePage;
