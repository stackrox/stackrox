import React, { useContext, useState } from 'react';
import {
    Divider,
    MenuToggle,
    MenuToggleElement,
    Select,
    SelectGroup,
    SelectList,
    SelectOption,
    Spinner,
} from '@patternfly/react-core';

import { ScanConfigurationsContext } from '../ScanConfigurationsProvider';

const ALL_SCAN_SCHEDULES_OPTION = 'All scan schedules';

function ScanConfigurationSelect() {
    const [isOpen, setIsOpen] = useState(false);

    const { scanConfigurationsQuery, selectedScanConfigName, setSelectedScanConfigName } =
        useContext(ScanConfigurationsContext);

    const onToggleClick = () => {
        setIsOpen(!isOpen);
    };

    const onSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        const selectedValue = value === ALL_SCAN_SCHEDULES_OPTION ? undefined : (value as string);
        setSelectedScanConfigName(selectedValue);
        setIsOpen(false);
    };

    const renderToggle = (toggleRef: React.Ref<HTMLButtonElement | MenuToggleElement>) => {
        return (
            <MenuToggle ref={toggleRef} onClick={onToggleClick} isExpanded={isOpen}>
                {selectedScanConfigName || ALL_SCAN_SCHEDULES_OPTION}
            </MenuToggle>
        );
    };

    return (
        <Select
            id="scan-schedules-filter-id"
            isOpen={isOpen}
            selected={selectedScanConfigName || ALL_SCAN_SCHEDULES_OPTION}
            onSelect={onSelect}
            onOpenChange={(isOpen) => setIsOpen(isOpen)}
            toggle={renderToggle}
            shouldFocusToggleOnSelect
        >
            <>
                <SelectGroup label="View all results">
                    <SelectList>
                        <SelectOption value={ALL_SCAN_SCHEDULES_OPTION}>
                            {ALL_SCAN_SCHEDULES_OPTION}
                        </SelectOption>
                    </SelectList>
                </SelectGroup>
                <Divider />
                <SelectGroup label="Filter results by a schedule">
                    <SelectList>
                        {scanConfigurationsQuery.isLoading ? (
                            <SelectOption isLoading value="loader" isDisabled>
                                <Spinner size="lg" />
                            </SelectOption>
                        ) : (
                            <>
                                {scanConfigurationsQuery.response.configurations.map(
                                    ({ scanName }) => (
                                        <SelectOption key={scanName} value={scanName}>
                                            {scanName}
                                        </SelectOption>
                                    )
                                )}
                            </>
                        )}
                    </SelectList>
                </SelectGroup>
            </>
        </Select>
    );
}

export default ScanConfigurationSelect;
