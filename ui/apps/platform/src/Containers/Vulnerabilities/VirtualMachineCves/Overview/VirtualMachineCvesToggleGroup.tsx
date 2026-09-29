import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import useURLPagination from 'hooks/useURLPagination';
import useURLParameter from 'hooks/useURLParameter';
import useURLStringUnion from 'hooks/useURLStringUnion';

import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import { virtualMachineEntityTabValues } from '../../types';
import type { VirtualMachineEntityTab } from '../../types';

// Shared entity toggle for both overview tables, which each render their own toolbar.
// Factored out to make sure the two tables stay in sync. State is read from the URL, not props.
function VirtualMachineCvesToggleGroup() {
    const [activeEntityTabKey, setActiveEntityTabKey] = useURLStringUnion(
        'entityTab',
        virtualMachineEntityTabValues
    );
    const { setPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const [, setSortOption] = useURLParameter('sortOption', undefined);

    function selectEntityTab(entityTab: VirtualMachineEntityTab) {
        setActiveEntityTabKey(entityTab);
        setPage(1);
        setSortOption(undefined);
    }

    return (
        <ToggleGroup aria-label="Entity type toggle items">
            <ToggleGroupItem
                text="CVEs"
                buttonId="CVE"
                isSelected={activeEntityTabKey === 'CVE'}
                onChange={() => selectEntityTab('CVE')}
            />
            <ToggleGroupItem
                text="Virtual Machines"
                buttonId="VirtualMachine"
                isSelected={activeEntityTabKey === 'VirtualMachine'}
                onChange={() => selectEntityTab('VirtualMachine')}
            />
        </ToggleGroup>
    );
}

export default VirtualMachineCvesToggleGroup;
