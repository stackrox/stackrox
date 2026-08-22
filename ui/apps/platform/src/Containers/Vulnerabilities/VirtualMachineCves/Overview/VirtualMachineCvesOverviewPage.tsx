import { useState } from 'react';
import {
    Content,
    Flex,
    PageSection,
    Title,
    ToggleGroup,
    ToggleGroupItem,
    ToolbarItem,
} from '@patternfly/react-core';

import ColumnManagementButton from 'Components/ColumnManagementButton';
import PageTitle from 'Components/PageTitle';
import TechnologyPreviewLabel from 'Components/PatternFly/PreviewLabel/TechnologyPreviewLabel';
import { useManagedColumns } from 'hooks/useManagedColumns';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { getHasSearchApplied } from 'utils/searchUtils';

import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import TableEntityToolbar from '../../components/TableEntityToolbar';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import {
    virtualMachineCVESearchFilterConfig,
    virtualMachineComponentSearchFilterConfig,
    virtualMachinesClusterSearchFilterConfig,
    virtualMachinesNamespaceSearchFilterConfig,
    virtualMachinesSearchFilterConfig,
} from '../../searchFilterConfig';
import { virtualMachineEntityTabValues } from '../../types';
import VirtualMachineScanScopeAlert from '../components/VirtualMachineScanScopeAlert';
import VirtualMachineCVEsTable from './VirtualMachineCVEsTable';
import VirtualMachinesCvesTable, { defaultColumns } from './VirtualMachinesCvesTable';

const searchFilterConfig = [
    virtualMachinesClusterSearchFilterConfig,
    virtualMachineCVESearchFilterConfig,
    virtualMachinesNamespaceSearchFilterConfig,
    virtualMachinesSearchFilterConfig,
    virtualMachineComponentSearchFilterConfig,
];

function VirtualMachineCvesOverviewPage() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isEnhancedDataModelEnabled = isFeatureFlagEnabled(
        'ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL'
    );
    const [activeEntityTabKey, setActiveEntityTabKey] = useURLStringUnion(
        'entityTab',
        virtualMachineEntityTabValues
    );
    const { searchFilter, setSearchFilter } = useURLSearch();
    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const [tableRowCount, setTableRowCount] = useState(0);
    const managedColumnState = useManagedColumns('VirtualMachinesCvesTable', defaultColumns);

    const isFiltered = getHasSearchApplied(searchFilter);

    function onEntityTabChange() {
        pagination.setPage(1);
        setTableRowCount(0);
    }

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
                {isEnhancedDataModelEnabled && (
                    <TableEntityToolbar
                        filterToolbar={
                            <AdvancedFiltersToolbar
                                searchFilter={searchFilter}
                                searchFilterConfig={searchFilterConfig}
                                defaultSearchFilterEntity="Virtual machine"
                                includeCveSeverityFilters={false}
                                includeCveStatusFilters={false}
                                onFilterChange={(newFilter) => {
                                    setSearchFilter(newFilter);
                                    pagination.setPage(1, 'replace');
                                }}
                            />
                        }
                        entityToggleGroup={
                            <ToggleGroup aria-label="Entity type toggle items">
                                <ToggleGroupItem
                                    text="CVEs"
                                    buttonId="CVE"
                                    isSelected={activeEntityTabKey === 'CVE'}
                                    onChange={() => {
                                        setActiveEntityTabKey('CVE');
                                        onEntityTabChange();
                                    }}
                                />
                                <ToggleGroupItem
                                    text="Virtual Machines"
                                    buttonId="VirtualMachine"
                                    isSelected={activeEntityTabKey === 'VirtualMachine'}
                                    onChange={() => {
                                        setActiveEntityTabKey('VirtualMachine');
                                        onEntityTabChange();
                                    }}
                                />
                            </ToggleGroup>
                        }
                        pagination={pagination}
                        tableRowCount={tableRowCount}
                        isFiltered={isFiltered}
                    >
                        {activeEntityTabKey === 'VirtualMachine' && (
                            <ToolbarItem>
                                <ColumnManagementButton
                                    columnConfig={managedColumnState.columns}
                                    onApplyColumns={managedColumnState.setVisibility}
                                />
                            </ToolbarItem>
                        )}
                    </TableEntityToolbar>
                )}
                {isEnhancedDataModelEnabled && activeEntityTabKey === 'CVE' && (
                    <VirtualMachineCVEsTable
                        searchFilter={searchFilter}
                        pagination={pagination}
                        onTotalCountChange={setTableRowCount}
                        onClearFilters={onClearFilters}
                    />
                )}
                {(!isEnhancedDataModelEnabled || activeEntityTabKey === 'VirtualMachine') && (
                    <VirtualMachinesCvesTable
                        searchFilter={searchFilter}
                        pagination={pagination}
                        managedColumnState={managedColumnState}
                        onTotalCountChange={setTableRowCount}
                        onClearFilters={onClearFilters}
                    />
                )}
            </PageSection>
        </>
    );
}

export default VirtualMachineCvesOverviewPage;
