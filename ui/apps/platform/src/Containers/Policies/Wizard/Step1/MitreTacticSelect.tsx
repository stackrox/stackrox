import React, { ReactElement } from 'react';
import {
    Select,
    MenuToggle,
    MenuToggleElement,
    SelectList,
    SelectOption,
    Flex,
    FlexItem,
} from '@patternfly/react-core';

import { MitreAttackVector } from 'types/mitre.proto';
import useSelectState from 'Components/SelectSingle/useSelectState';

type MitreTacticSelectProps = {
    className: string;
    getIsDisabledOption: (optionId: string) => boolean;
    handleSelectOption: (optionId: string) => void;
    isDisabled: boolean;
    label: string;
    mitreAttackVectors: MitreAttackVector[];
    tacticId: string;
};

/*
 * Select to add or replace a MITRE ATT&CK tactic.
 */
function MitreTacticSelect({
    className,
    getIsDisabledOption, // is tactic already selected
    handleSelectOption,
    isDisabled, // Select element is disabled if tactic has techniques
    label,
    mitreAttackVectors,
    tacticId,
}: MitreTacticSelectProps): ReactElement {
    const { isOpen, setIsOpen, onSelect, onToggle } = useSelectState(handleSelectOption);

    // Find the display content for the selected value
    const getDisplayContent = (): React.ReactNode => {
        if (!tacticId) {
            return label;
        }

        const selectedTactic = mitreAttackVectors.find(({ tactic }) => tactic.id === tacticId);
        if (selectedTactic) {
            return (
                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                    <FlexItem className="pf-v5-u-text-truncate">
                        {selectedTactic.tactic.name}
                    </FlexItem>
                    <FlexItem className="pf-v5-u-color-200 pf-v5-u-font-size-sm">|</FlexItem>
                    <FlexItem className="pf-v5-u-color-200 pf-v5-u-font-size-sm">
                        {selectedTactic.tactic.id}
                    </FlexItem>
                </Flex>
            );
        }

        return tacticId;
    };

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={onToggle}
            isExpanded={isOpen}
            isDisabled={isDisabled}
            aria-label={label}
            className="pf-v5-u-w-100"
        >
            {getDisplayContent()}
        </MenuToggle>
    );

    return (
        <Select
            aria-label={label}
            className={className}
            isOpen={isOpen}
            selected={tacticId}
            onSelect={onSelect}
            onOpenChange={(nextOpen: boolean) => setIsOpen(nextOpen)}
            toggle={toggle}
            shouldFocusToggleOnSelect
        >
            <SelectList style={{ maxHeight: '300px', overflowY: 'auto' }}>
                {mitreAttackVectors.map(({ tactic: { id, name } }) => {
                    return (
                        <SelectOption
                            key={id}
                            value={id}
                            isDisabled={getIsDisabledOption(id)}
                            description={id}
                        >
                            {name}
                        </SelectOption>
                    );
                })}
            </SelectList>
        </Select>
    );
}

export default MitreTacticSelect;
