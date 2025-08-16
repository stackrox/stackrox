import React, { ReactElement } from 'react';
import {
    Select,
    MenuToggle,
    MenuToggleElement,
    SelectList,
    SelectOption,
    Divider,
    Flex,
    FlexItem,
} from '@patternfly/react-core';

import { MitreTechnique } from 'types/mitre.proto';
import useSelectState from 'Components/SelectSingle/useSelectState';

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
    const { isOpen, setIsOpen, onSelect, onToggle } = useSelectState(handleSelectOption);

    // Group techniques by base technique ID
    const groupedTechniques = mitreTechniques.reduce(
        (groups, technique) => {
            const baseId = technique.id.includes('.') ? technique.id.split('.')[0] : technique.id;
            const updatedGroups = { ...groups };
            if (!updatedGroups[baseId]) {
                updatedGroups[baseId] = [];
            }
            updatedGroups[baseId] = [...updatedGroups[baseId], technique];
            return updatedGroups;
        },
        {} as Record<string, MitreTechnique[]>
    );

    // Find the display content for the selected value
    const getDisplayContent = (): React.ReactNode => {
        if (!techniqueId) {
            return label;
        }

        const selectedTechnique = mitreTechniques.find((technique) => technique.id === techniqueId);
        if (selectedTechnique) {
            const indexOfColonSpace = selectedTechnique.name.indexOf(': ');
            const displayName =
                indexOfColonSpace === -1
                    ? selectedTechnique.name
                    : selectedTechnique.name.slice(indexOfColonSpace + 2);

            return (
                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                    <FlexItem className="pf-v5-u-text-truncate">{displayName}</FlexItem>
                    <FlexItem className="pf-v5-u-color-200 pf-v5-u-font-size-sm">|</FlexItem>
                    <FlexItem className="pf-v5-u-color-200 pf-v5-u-font-size-sm">
                        {selectedTechnique.id}
                    </FlexItem>
                </Flex>
            );
        }

        return techniqueId;
    };

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={onToggle}
            isExpanded={isOpen}
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
            selected={techniqueId}
            onSelect={onSelect}
            onOpenChange={(nextOpen: boolean) => setIsOpen(nextOpen)}
            toggle={toggle}
            shouldFocusToggleOnSelect
        >
            <SelectList style={{ maxHeight: '300px', overflowY: 'auto' }}>
                {Object.entries(groupedTechniques).map(([baseId, techniques], index, array) => {
                    // Sort techniques within group: base technique first, then sub-techniques
                    const sortedTechniques = techniques.sort((a, b) => {
                        if (a.id === baseId) {
                            return -1;
                        }
                        if (b.id === baseId) {
                            return 1;
                        }
                        return a.id.localeCompare(b.id);
                    });

                    const isLastGroup = index === array.length - 1;

                    return (
                        <React.Fragment key={baseId}>
                            {sortedTechniques.map(({ id, name }) => {
                                const indexOfColonSpace = name.indexOf(': ');
                                const displayName =
                                    indexOfColonSpace === -1
                                        ? name
                                        : name.slice(indexOfColonSpace + 2);

                                return (
                                    <SelectOption
                                        key={id}
                                        value={id}
                                        isDisabled={getIsDisabledOption(id)}
                                        description={id}
                                    >
                                        {displayName}
                                    </SelectOption>
                                );
                            })}
                            {!isLastGroup && <Divider component="li" />}
                        </React.Fragment>
                    );
                })}
            </SelectList>
        </Select>
    );
}

export default MitreTechniqueSelect;
