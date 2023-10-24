import React, { useState } from 'react';
import { Divider, Select, SelectOption } from '@patternfly/react-core';

import { AdministrationEventLevel, levels } from 'services/AdministrationEventsService';

import { getLevelText } from './AdministrationEvent';

const optionAll = 'All';

type SearchFilterLevelProps = {
    isDisabled: boolean;
    level: AdministrationEventLevel | undefined;
    setLevel: (level: AdministrationEventLevel) => void;
};

function SearchFilterLevel({ isDisabled, level, setLevel }: SearchFilterLevelProps) {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(_event, selection) {
        setLevel(selection === optionAll ? undefined : selection);
        setIsOpen(false);
    }

    const options = levels.map((levelArg) => (
        <SelectOption key={levelArg} value={levelArg}>
            {getLevelText(levelArg)}
        </SelectOption>
    ));
    options.push(
        <Divider key="Divider" />,
        <SelectOption key="All" value={optionAll} isPlaceholder>
            All levels
        </SelectOption>
    );

    return (
        <Select
            variant="single"
            aria-label="Level filter menu items"
            toggleAriaLabel="Level filter menu toggle"
            onToggle={setIsOpen}
            onSelect={onSelect}
            selections={level ?? optionAll}
            isDisabled={isDisabled}
            isOpen={isOpen}
        >
            {options}
        </Select>
    );
}

export default SearchFilterLevel;
