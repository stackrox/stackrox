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

import useFeatureFlags from 'hooks/useFeatureFlags';

import { ObservedCveMode, isObservedCveMode, observedCveModeValues } from '../../types';
import { getViewStateDescription, getViewStateTitle } from './string.utils';

const width = '330px';

export type ObservedCveModeSelectProps = {
    observedCveMode: ObservedCveMode;
    setObservedCveMode: (value: ObservedCveMode) => void;
};

function ObservedCveModeSelect({
    observedCveMode,
    setObservedCveMode,
}: ObservedCveModeSelectProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    if (isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')) {
        // Delete this component when the above feature flag is removed
    }

    const [isCveModeSelectOpen, setIsCveModeSelectOpen] = useState(false);

    const isViewingWithCves = observedCveMode === 'WITH_CVES';

    const menuToggleIcon = isViewingWithCves ? <SecurityIcon /> : <UnknownIcon />;

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
                    style={{ width }}
                    aria-label="Observed CVE mode select"
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
            <SelectList style={{ width }}>
                <SelectOption
                    value={observedCveModeValues[0]}
                    description={getViewStateDescription('OBSERVED', true)}
                >
                    {getViewStateTitle('OBSERVED', true)}
                </SelectOption>
                <SelectOption
                    value={observedCveModeValues[1]}
                    description={getViewStateDescription('OBSERVED', false)}
                >
                    {getViewStateTitle('OBSERVED', false)}
                </SelectOption>
            </SelectList>
        </Select>
    );
}

export default ObservedCveModeSelect;
