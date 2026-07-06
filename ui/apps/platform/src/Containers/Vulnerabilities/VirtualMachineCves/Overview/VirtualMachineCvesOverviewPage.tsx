import {
    Content,
    Flex,
    PageSection,
    Title,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import TechnologyPreviewLabel from 'Components/PatternFly/PreviewLabel/TechnologyPreviewLabel';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useURLStringUnion from 'hooks/useURLStringUnion';

import { virtualMachineEntityTabValues } from '../../types';
import VirtualMachineScanScopeAlert from '../components/VirtualMachineScanScopeAlert';
import VirtualMachineCVEsTable from './VirtualMachineCVEsTable';
import VirtualMachinesCvesTable from './VirtualMachinesCvesTable';

function VirtualMachineCvesOverviewPage() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isEnhancedDataModelEnabled = isFeatureFlagEnabled(
        'ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL'
    );
    const [activeEntityTabKey, setActiveEntityTabKey] = useURLStringUnion(
        'entityTab',
        virtualMachineEntityTabValues
    );

    return (
        <>
            <PageTitle title="Virtual Machine CVEs Overview" />
            <PageSection component="div">
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <Title headingLevel="h1">Virtual machine vulnerabilities</Title>
                        <TechnologyPreviewLabel />
                    </Flex>
                    <Content component="p">
                        Prioritize and remediate observed CVEs across virtual machines
                    </Content>
                </Flex>
            </PageSection>
            <PageSection hasBodyWrapper={false}>
                <VirtualMachineScanScopeAlert />
                {isEnhancedDataModelEnabled && (
                    <ToggleGroup aria-label="Entity type toggle items">
                        <ToggleGroupItem
                            text="CVEs"
                            buttonId="CVE"
                            isSelected={activeEntityTabKey === 'CVE'}
                            onChange={() => setActiveEntityTabKey('CVE')}
                        />
                        <ToggleGroupItem
                            text="Virtual Machines"
                            buttonId="VirtualMachine"
                            isSelected={activeEntityTabKey === 'VirtualMachine'}
                            onChange={() => setActiveEntityTabKey('VirtualMachine')}
                        />
                    </ToggleGroup>
                )}
                {isEnhancedDataModelEnabled && activeEntityTabKey === 'CVE' && (
                    <VirtualMachineCVEsTable />
                )}
                {(!isEnhancedDataModelEnabled || activeEntityTabKey === 'VirtualMachine') && (
                    <VirtualMachinesCvesTable />
                )}
            </PageSection>
        </>
    );
}

export default VirtualMachineCvesOverviewPage;
