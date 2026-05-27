import { useState } from 'react';
import type { MouseEvent as ReactMouseEvent, Ref } from 'react';
import {
    Button,
    Divider,
    Flex,
    Label,
    MenuToggle,
    Select,
    SelectGroup,
    SelectList,
    SelectOption,
    Spinner,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import { TimesCircleIcon } from '@patternfly/react-icons';

import type { ComplianceScanConfigOverview } from 'services/ComplianceScanConfigurationService';

const ALL_SCAN_SCHEDULES_OPTION = 'All scan schedules';

type ScanConfigurationSelectProps = {
    isLoading: boolean;
    scanConfigs: ComplianceScanConfigOverview[];
    selectedScanConfigName: string | undefined;
    isScanConfigDisabled?: (config: ComplianceScanConfigOverview) => boolean;
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

    const managedConfigs = scanConfigs.filter((c) => c.isManaged);
    const discoveredConfigs = scanConfigs.filter((c) => !c.isManaged);

    const onToggleClick = () => {
        setIsOpen((prev) => !prev);
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
        <Flex justifyContent={{ default: 'justifyContentSpaceBetween' }}>
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
                    {isLoading ? (
                        <SelectGroup label="Filter results by a schedule">
                            <SelectList>
                                <SelectOption isLoading value="loader" isDisabled>
                                    <Spinner size="lg" />
                                </SelectOption>
                            </SelectList>
                        </SelectGroup>
                    ) : (
                        <>
                            {managedConfigs.length > 0 && (
                                <SelectGroup label="Managed scan schedules">
                                    <SelectList>
                                        {managedConfigs.map((config) => (
                                            <SelectOption
                                                key={config.scanConfigName}
                                                value={config.scanConfigName}
                                                isDisabled={isScanConfigDisabled(config)}
                                            >
                                                {config.scanConfigName}
                                            </SelectOption>
                                        ))}
                                    </SelectList>
                                </SelectGroup>
                            )}
                            {discoveredConfigs.length > 0 && (
                                <>
                                    {managedConfigs.length > 0 && <Divider />}
                                    <SelectGroup label="Discovered scan schedules">
                                        <SelectList>
                                            {discoveredConfigs.map((config) => (
                                                <SelectOption
                                                    key={config.scanConfigName}
                                                    value={config.scanConfigName}
                                                    isDisabled={isScanConfigDisabled(config)}
                                                    description="External"
                                                >
                                                    {config.scanConfigName}{' '}
                                                    <Label
                                                        isCompact
                                                        color="blue"
                                                    >
                                                        External
                                                    </Label>
                                                </SelectOption>
                                            ))}
                                        </SelectList>
                                    </SelectGroup>
                                </>
                            )}
                        </>
                    )}
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
