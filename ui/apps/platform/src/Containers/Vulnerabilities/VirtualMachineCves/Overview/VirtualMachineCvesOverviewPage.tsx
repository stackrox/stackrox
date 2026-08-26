import { Content, Flex, PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import TechnologyPreviewLabel from 'Components/PatternFly/PreviewLabel/TechnologyPreviewLabel';
import { useManagedColumns } from 'hooks/useManagedColumns';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { getHasSearchApplied } from 'utils/searchUtils';

import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import { virtualMachineEntityTabValues } from '../../types';
import VirtualMachineScanScopeAlert from '../components/VirtualMachineScanScopeAlert';
import VirtualMachineCvesCveTable from './VirtualMachineCvesCveTable';
import VirtualMachineCvesVmTable, { defaultColumns } from './VirtualMachineCvesVmTable';

function VirtualMachineCvesOverviewPage() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isEnhancedDataModelEnabled = isFeatureFlagEnabled(
        'ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL'
    );
    const [activeEntityTabKey] = useURLStringUnion('entityTab', virtualMachineEntityTabValues);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const managedColumnState = useManagedColumns('VirtualMachinesCvesTable', defaultColumns);

    const isFiltered = getHasSearchApplied(searchFilter);

    function onClearFilters() {
        setSearchFilter({});
        pagination.setPage(1);
    }

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
            <PageSection>
                <VirtualMachineScanScopeAlert />
            </PageSection>
            <PageSection isCenterAligned>
                {isEnhancedDataModelEnabled && activeEntityTabKey === 'CVE' && (
                    <VirtualMachineCvesCveTable
                        searchFilter={searchFilter}
                        pagination={pagination}
                        isFiltered={isFiltered}
                        onClearFilters={onClearFilters}
                    />
                )}
                {(!isEnhancedDataModelEnabled || activeEntityTabKey === 'VirtualMachine') && (
                    <VirtualMachineCvesVmTable
                        searchFilter={searchFilter}
                        pagination={pagination}
                        managedColumnState={managedColumnState}
                        isFiltered={isFiltered}
                        onClearFilters={onClearFilters}
                    />
                )}
            </PageSection>
        </>
    );
}

export default VirtualMachineCvesOverviewPage;
