import React, { useState } from 'react';
import type { MouseEvent as ReactMouseEvent, Ref } from 'react';
import {
    Button,
    Divider,
    Flex,
    MenuToggle,
    Select,
    SelectGroup,
    SelectList,
    SelectOption,
    Spinner,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import { TimesCircleIcon } from '@patternfly/react-icons';

import type { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';

const ALL_SCAN_SCHEDULES_OPTION = 'All scan schedules';

type ScanConfigurationSelectProps = {
    isLoading: boolean;
    scanConfigs: ComplianceScanConfigurationStatus[];
    selectedScanConfigName: string | undefined;
    isScanConfigDisabled?: (config: ComplianceScanConfigurationStatus) => boolean;
    setSelectedScanConfigName: (value: string | undefined) => void;
};

function ScanConfigurationSelect({
    isLoading,
    scanConfigs,
    selectedScanConfigName,
    isScanConfigDisabled = () => false,
    setSelectedScanConfigName,
}: ScanConfigurationSelectProps) {
    const [isOpen, setIsOpen] = useState(false);

    const onToggleClick = () => {
        setIsOpen(!isOpen);
    };

    const onSelect = (
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        const selectedValue = value === ALL_SCAN_SCHEDULES_OPTION ? undefined : (value as string);
        setSelectedScanConfigName(selectedValue);
        setIsOpen(false);
    };

    const renderToggle = (toggleRef: Ref<HTMLButtonElement | MenuToggleElement>) => {
        return (
            <MenuToggle ref={toggleRef} onClick={onToggleClick} isExpanded={isOpen}>
                {selectedScanConfigName || ALL_SCAN_SCHEDULES_OPTION}
            </MenuToggle>
        );
    };

    return (
        <Flex
            className="pf-v5-u-px-lg pf-v5-u-py-sm"
            justifyContent={{ default: 'justifyContentSpaceBetween' }}
        >
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
                            {isLoading ? (
                                <SelectOption isLoading value="loader" isDisabled>
                                    <Spinner size="lg" />
                                </SelectOption>
                            ) : (
                                <>
                                    {scanConfigs.map((config) => {
                                        return (
                                            <SelectOption
                                                key={config.id}
                                                value={config.scanName}
                                                isDisabled={isScanConfigDisabled(config)}
                                            >
                                                {config.scanName}
                                            </SelectOption>
                                        );
                                    })}
                                </>
                            )}
                        </SelectList>
                    </SelectGroup>
                </>
            </Select>
            <Button
                variant="link"
                icon={<TimesCircleIcon />}
                isDisabled={!selectedScanConfigName}
                onClick={() => setSelectedScanConfigName(undefined)}
            >
                Reset filter
            </Button>
        </Flex>
    );
}

export default ScanConfigurationSelect;
