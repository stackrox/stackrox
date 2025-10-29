import { Divider, SelectOption } from '@patternfly/react-core';
import SelectSingle from 'Components/SelectSingle/SelectSingle';

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

    function onSelect(_id: string, selection: string) {
        if (selection === optionAll) {
            setLevel(undefined);
        } else {
            const selectedLevel = levels.find((lv) => getLevelText(lv) === selection);
            setLevel(selectedLevel || undefined);
        }
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
        <SelectSingle
            id="level-filter"
            value={displayValue}
            handleSelect={onSelect}
            isDisabled={isDisabled}
            placeholderText="Select level"
            toggleAriaLabel="Level filter menu toggle"
        >
            {options}
        </SelectSingle>
    );
}

export default SearchFilterLevel;
