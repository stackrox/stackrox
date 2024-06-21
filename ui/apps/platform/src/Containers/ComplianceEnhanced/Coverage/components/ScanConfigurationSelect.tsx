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

    const { scanConfigurationsQuery, selectedScanConfig, setSelectedScanConfig } =
        useContext(ScanConfigurationsContext);

    const onToggleClick = () => {
        setIsOpen(!isOpen);
    };

    const onSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        const selectedValue = value === ALL_SCAN_SCHEDULES_OPTION ? undefined : (value as string);
        setSelectedScanConfig(selectedValue);
        setIsOpen(false);
    };

    const renderToggle = (toggleRef: React.Ref<HTMLButtonElement | MenuToggleElement>) => {
        return (
            <MenuToggle ref={toggleRef} onClick={onToggleClick} isExpanded={isOpen}>
                {(selectedScanConfig as string) || ALL_SCAN_SCHEDULES_OPTION}
            </MenuToggle>
        );
    };

    return (
        <Select
            id="scan-schedules-filter-id"
            isOpen={isOpen}
            selected={selectedScanConfig || ALL_SCAN_SCHEDULES_OPTION}
            onSelect={onSelect}
            onOpenChange={(isOpen) => setIsOpen(isOpen)}
            toggle={renderToggle}
            shouldFocusToggleOnSelect
        >
            {scanConfigurationsQuery.isLoading ? (
                <SelectOption isLoading value="loader">
                    <Spinner size="lg" />
                </SelectOption>
            ) : (
                <>
                    <SelectGroup label="View all results">
                        <SelectList>
                            <SelectOption
                                key="key_all-scan-schedules"
                                value={ALL_SCAN_SCHEDULES_OPTION}
                            >
                                {ALL_SCAN_SCHEDULES_OPTION}
                            </SelectOption>
                        </SelectList>
                    </SelectGroup>
                    <Divider />
                    <SelectGroup label="Filter results by a schedule">
                        <SelectList>
                            {scanConfigurationsQuery.response.configurations.map(({ scanName }) => (
                                <SelectOption key={scanName} value={scanName}>
                                    {scanName}
                                </SelectOption>
                            ))}
                        </SelectList>
                    </SelectGroup>
                </>
            )}
        </Select>
    );
}

export default ScanConfigurationSelect;
