import React, { ReactElement, useState } from 'react';
import { Flex, Select, SelectOption } from '@patternfly/react-core';

import { MitreTechnique } from 'types/mitre.proto';

type MitreTechniqueSelectProps = {
    className: string;
    getIsDisabledOption: (optionId: string) => boolean;
    handleSelectOption: (optionId: string) => void;
    label: string;
    mitreTechniques: MitreTechnique[];
    techniqueId: string;
};

/*
 * Select to either add or replace a MITRE ATT&CK technique for a tactic.
 */
function MitreTechniqueSelect({
    className,
    getIsDisabledOption, // is technique already selected for tactic
    handleSelectOption,
    label,
    mitreTechniques, // relevant techniques for tactic
    techniqueId,
}: MitreTechniqueSelectProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(_event, optionId) {
        setIsOpen(false);
        handleSelectOption(optionId);
    }

    return (
        <Select
            aria-label={label}
            className={className}
            isOpen={isOpen}
            onSelect={onSelect}
            onToggle={setIsOpen}
            placeholderText={label}
            selections={techniqueId}
        >
            {mitreTechniques.map(({ id, name }) => {
                const optionClassName = id.includes('.')
                    ? 'mitre-subtechnique-option'
                    : 'mitre-technique-option';
                const indexOfColonSpace = name.indexOf(': ');

                // See MitreAttackVectorsFormSection.css for name and id style rules.
                return (
                    <SelectOption
                        key={id}
                        value={id}
                        isDisabled={getIsDisabledOption(id)}
                        className={optionClassName}
                    >
                        <Flex
                            flexWrap={{ default: 'nowrap' }}
                            justifyContent={{ default: 'justifyContentSpaceBetween' }}
                        >
                            <span className="name">
                                {indexOfColonSpace === -1
                                    ? name
                                    : name.slice(indexOfColonSpace + 2)}
                            </span>
                            <span className="id">{id}</span>
                        </Flex>
                    </SelectOption>
                );
            })}
        </Select>
    );
}

export default MitreTechniqueSelect;
