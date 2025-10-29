import { Divider } from '@patternfly/react-core';
import { SelectOption } from '@patternfly/react-core';
import SimpleSelect from 'Components/CompoundSearchFilter/components/SimpleSelect';

import { levels } from 'services/AdministrationEventsService';
import type { AdministrationEventLevel } from 'services/AdministrationEventsService';

import { getLevelText } from './AdministrationEvent';

const optionAll = 'All levels';

type SearchFilterLevelProps = {
    isDisabled: boolean;
    level: AdministrationEventLevel | undefined;
    setLevel: (level: AdministrationEventLevel | undefined) => void;
};

function SearchFilterLevel({ isDisabled, level, setLevel }: SearchFilterLevelProps) {
    const displayValue = level ? getLevelText(level) : optionAll;

    function onSelect(selection: string | number | undefined) {
        const selectedLevel = levels.find((lv) => getLevelText(lv) === selection);
        setLevel(selectedLevel || undefined);
    }

    const options = levels.map((levelArg) => (
        <SelectOption key={levelArg} value={getLevelText(levelArg)}>
            {getLevelText(levelArg)}
        </SelectOption>
    ));
    options.push(
        <Divider key="Divider" />,
        <SelectOption key="All" value={optionAll}>
            {optionAll}
        </SelectOption>
    );

    return (
        <SimpleSelect
            value={displayValue}
            onChange={onSelect}
            isDisabled={isDisabled}
            ariaLabelMenu="Level filter menu items"
            ariaLabelToggle="Level filter menu toggle"
        >
            {options}
        </SimpleSelect>
    );
}

export default SearchFilterLevel;
