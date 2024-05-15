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

export const WITH_CVE_OPTION_TITLE = 'Image vulnerabilities';
export const WITHOUT_CVE_OPTION_TITLE = 'Images without vulnerabilities';
export const WITH_CVE_OPTION_DESCRIPTION = 'Images and deployments observed with CVEs';
export const WITHOUT_CVE_OPTION_DESCRIPTION =
    'Images and deployments observed without CVEs (results may be inaccurate due to scanner errors)';

export type ObservedCveModeSelectProps = {
    observedCveMode: ObservedCveMode;
    setObservedCveMode: (value: ObservedCveMode) => void;
};

function ObservedCveModeSelect({
    observedCveMode,
    setObservedCveMode,
}: ObservedCveModeSelectProps) {
    const [isCveModeSelectOpen, setIsCveModeSelectOpen] = useState(false);
    const isViewingWithCves = observedCveMode === 'WITH_CVES';

    const menuToggleIcon = isViewingWithCves ? (
        <SecurityIcon color={CRITICAL_SEVERITY_COLOR} />
    ) : (
        <UnknownIcon />
    );

    const menuToggleText = isViewingWithCves
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
                    description={WITH_CVE_OPTION_DESCRIPTION}
                >
                    {WITH_CVE_OPTION_TITLE}
                </SelectOption>
                <SelectOption
                    value={observedCveModeValues[1]}
                    description={WITHOUT_CVE_OPTION_DESCRIPTION}
                >
                    {WITHOUT_CVE_OPTION_TITLE}
                </SelectOption>
            </SelectList>
        </Select>
    );
}

export default ObservedCveModeSelect;
