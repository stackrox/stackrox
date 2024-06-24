import React, { useState } from 'react';
import {
    Button,
    Divider,
    Flex,
    MenuToggle,
    MenuToggleElement,
    Select,
    SelectGroup,
    SelectList,
    SelectOption,
    Spinner,
} from '@patternfly/react-core';
import { TimesCircleIcon } from '@patternfly/react-icons';

const ALL_SCAN_SCHEDULES_OPTION = 'All scan schedules';

export type ScanConfigurationSelectData = {
    id: string;
    isDisabled: boolean;
    name: string;
};

type ScanConfigurationSelectProps = {
    isLoading: boolean;
    scanConfigs: ScanConfigurationSelectData[];
    selectedScanConfigName: string | undefined;
    setSelectedScanConfigName: (value: string | undefined) => void;
};

function ScanConfigurationSelect({
    isLoading,
    scanConfigs,
    selectedScanConfigName,
    setSelectedScanConfigName,
}: ScanConfigurationSelectProps) {
    const [isOpen, setIsOpen] = useState(false);

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
                                                value={config.name}
                                                isDisabled={config.isDisabled}
                                            >
                                                {config.name}
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
                onClick={() => setSelectedScanConfigName(undefined)}
            >
                Reset filter
            </Button>
        </Flex>
    );
}

export default ScanConfigurationSelect;
