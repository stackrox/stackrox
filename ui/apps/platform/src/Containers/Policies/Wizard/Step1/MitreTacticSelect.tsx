import React, { ReactElement, useState } from 'react';
import { Flex, Select, SelectOption } from '@patternfly/react-core';

import { MitreAttackVector } from 'types/mitre.proto';

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
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(_event, optionId) {
        setIsOpen(false);
        handleSelectOption(optionId);
    }

    return (
        <Select
            aria-label={label}
            className={className}
            isDisabled={isDisabled}
            isOpen={isOpen}
            onSelect={onSelect}
            onToggle={setIsOpen}
            placeholderText={label}
            selections={tacticId}
        >
            {mitreAttackVectors.map(({ tactic: { id, name } }) => {
                // See MitreAttackVectorsFormSection.css for name and id style rules.
                return (
                    <SelectOption
                        key={id}
                        value={id}
                        isDisabled={getIsDisabledOption(id)}
                        className="mitre-tactic-option"
                    >
                        <Flex
                            flexWrap={{ default: 'nowrap' }}
                            justifyContent={{ default: 'justifyContentSpaceBetween' }}
                        >
                            <span className="name">{name}</span>
                            <span className="id">{id}</span>
                        </Flex>
                    </SelectOption>
                );
            })}
        </Select>
    );
}

export default MitreTacticSelect;
