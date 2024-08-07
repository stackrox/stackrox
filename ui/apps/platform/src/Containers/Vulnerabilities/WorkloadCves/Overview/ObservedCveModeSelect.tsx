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
                    description={getViewStateDescription('OBSERVED', 'WITH_CVES')}
                >
                    {getViewStateTitle('OBSERVED', 'WITH_CVES')}
                </SelectOption>
                <SelectOption
                    value={observedCveModeValues[1]}
                    description={getViewStateDescription('OBSERVED', 'WITHOUT_CVES')}
                >
                    {getViewStateTitle('OBSERVED', 'WITHOUT_CVES')}
                </SelectOption>
            </SelectList>
        </Select>
    );
}

export default ObservedCveModeSelect;
