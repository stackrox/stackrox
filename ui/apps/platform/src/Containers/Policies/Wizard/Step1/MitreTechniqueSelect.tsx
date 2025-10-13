import React, { ReactElement } from 'react';
import {
    Select,
    MenuToggle,
    MenuToggleElement,
    SelectList,
    SelectOption,
    SelectGroup,
    Divider,
    Flex,
    FlexItem,
} from '@patternfly/react-core';

import { MitreTechnique } from 'types/mitre.proto';
import useSelectToggleState from 'Components/SelectSingle/useSelectToggleState';

type GroupedTechnique = {
    baseId: string;
    groupLabel: string;
    techniques: MitreTechnique[];
};

/**
 * Formats a technique name for display by removing the prefix before the colon
 */
export function formatTechniqueDisplayName(name: string): string {
    const indexOfColonSpace = name.indexOf(':');
    return indexOfColonSpace === -1 ? name : name.slice(indexOfColonSpace + 1).trim();
}

/**
 * Groups MITRE techniques by base ID and sorts them appropriately
 */
export function groupAndSortTechniques(techniques: MitreTechnique[]): GroupedTechnique[] {
    // Group techniques by base technique ID
    const techniqueGroups: Record<string, MitreTechnique[]> = {};
    techniques.forEach((technique) => {
        const baseId = technique.id.includes('.') ? technique.id.split('.')[0] : technique.id;
        if (!techniqueGroups[baseId]) {
            techniqueGroups[baseId] = [];
        }
        techniqueGroups[baseId].push(technique);
    });

    // Convert to array and sort each group
    return Object.entries(techniqueGroups).map(([baseId, groupTechniques]) => {
        // Sort techniques within group: base technique first, then sub-techniques
        const sortedTechniques = groupTechniques.sort((a, b) => {
            if (a.id === baseId) {
                return -1;
            }
            if (b.id === baseId) {
                return 1;
            }
            return a.id.localeCompare(b.id);
        });

        // Find the base technique for the group label
        const baseTechnique = sortedTechniques.find((technique) => technique.id === baseId);
        const groupLabel = baseTechnique ? baseTechnique.name : '';

        return {
            baseId,
            groupLabel,
            techniques: sortedTechniques,
        };
    });
}

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
    const { isOpen, setIsOpen, onSelect, onToggle } = useSelectToggleState(handleSelectOption);

    const groupedTechniques = groupAndSortTechniques(mitreTechniques);
    const selectedTechnique = mitreTechniques.find((technique) => technique.id === techniqueId);

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={onToggle}
            isExpanded={isOpen}
            aria-label={label}
            className="pf-v5-u-w-100"
        >
            {!techniqueId ? (
                label
            ) : selectedTechnique ? (
                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                    <FlexItem className="pf-v5-u-text-truncate">
                        {formatTechniqueDisplayName(selectedTechnique.name)}
                    </FlexItem>
                    <FlexItem className="pf-v5-u-color-200 pf-v5-u-font-size-sm">|</FlexItem>
                    <FlexItem className="pf-v5-u-color-200 pf-v5-u-font-size-sm">
                        {selectedTechnique.id}
                    </FlexItem>
                </Flex>
            ) : (
                techniqueId
            )}
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
                {groupedTechniques.map(({ baseId, groupLabel, techniques }, index) => {
                    const isLastGroup = index === groupedTechniques.length - 1;

                    return (
                        <React.Fragment key={baseId}>
                            <SelectGroup label={groupLabel}>
                                {techniques.map(({ id, name }) => (
                                    <SelectOption
                                        key={id}
                                        value={id}
                                        isDisabled={getIsDisabledOption(id)}
                                        description={id}
                                    >
                                        {formatTechniqueDisplayName(name)}
                                    </SelectOption>
                                ))}
                            </SelectGroup>
                            {!isLastGroup && <Divider component="li" />}
                        </React.Fragment>
                    );
                })}
            </SelectList>
        </Select>
    );
}

export default MitreTechniqueSelect;
