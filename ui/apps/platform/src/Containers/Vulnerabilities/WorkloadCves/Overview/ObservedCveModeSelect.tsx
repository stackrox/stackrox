import React, { useState } from 'react';
import {
    Flex,
    Icon,
    MenuToggleElement,
    MenuToggle,
    Select,
    SelectList,
    SelectOption,
} from '@patternfly/react-core';
import { SecurityIcon, UnknownIcon } from '@patternfly/react-icons';

import { CRITICAL_SEVERITY_COLOR } from 'constants/severityColors';
import { ObservedCveMode, isObservedCveMode, observedCveModeValues } from '../../types';

export type ObservedCveModeSelectProps = {
    observedCveMode: ObservedCveMode;
    setObservedCveMode: (value: ObservedCveMode) => void;
};

function ObservedCveModeSelect({
    observedCveMode,
    setObservedCveMode,
}: ObservedCveModeSelectProps) {
    const [isCveModeSelectOpen, setIsCveModeSelectOpen] = useState(false);

    const menuToggleIcon =
        observedCveMode === 'WITH_CVES' ? (
            <SecurityIcon color={CRITICAL_SEVERITY_COLOR} />
        ) : (
            <UnknownIcon />
        );

    const menuToggleText =
        observedCveMode === 'WITH_CVES'
            ? 'View image vulnerabilities'
            : 'View images without vulnerabilities';

    return (
        <Select
            isOpen={isCveModeSelectOpen}
            selected={observedCveMode}
            onSelect={(_, value) => {
                if (isObservedCveMode(value)) {
                    setObservedCveMode(value);
                    setIsCveModeSelectOpen(false);
                }
            }}
            onOpenChange={(isOpen) => setIsCveModeSelectOpen(isOpen)}
            toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                <MenuToggle
                    ref={toggleRef}
                    onClick={() => setIsCveModeSelectOpen(!isCveModeSelectOpen)}
                    isExpanded={isCveModeSelectOpen}
                >
                    <Flex
                        spaceItems={{ default: 'spaceItemsSm' }}
                        alignItems={{ default: 'alignItemsCenter' }}
                    >
                        <Icon>{menuToggleIcon}</Icon>
                        <span>{menuToggleText}</span>
                    </Flex>
                </MenuToggle>
            )}
            shouldFocusToggleOnSelect
        >
            <SelectList style={{ maxWidth: '300px' }}>
                <SelectOption
                    value={observedCveModeValues[0]}
                    description="Images and deployments observed with CVEs"
                >
                    Image vulnerabilities
                </SelectOption>
                <SelectOption
                    value={observedCveModeValues[1]}
                    description="Images observed without CVEs (results may be inaccurate due to scanner errors)"
                >
                    Images without vulnerabilities
                </SelectOption>
            </SelectList>
        </Select>
    );
}

export default ObservedCveModeSelect;
