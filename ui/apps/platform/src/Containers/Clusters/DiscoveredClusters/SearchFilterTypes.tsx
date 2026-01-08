import { Divider, SelectOption } from '@patternfly/react-core';

import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import { isType, types } from 'services/DiscoveredClusterService';
import type { DiscoveredClusterType } from 'services/DiscoveredClusterService';

import { getTypeText } from './DiscoveredCluster';

const optionAll = '##All Types##';

type SearchFilterTypesProps = {
    typesSelected: DiscoveredClusterType[] | undefined;
    isDisabled: boolean;
    setTypesSelected: (types: DiscoveredClusterType[] | undefined) => void;
};

function SearchFilterTypes({
    typesSelected,
    isDisabled,
    setTypesSelected,
}: SearchFilterTypesProps) {
    function onSelect(selections: string[]) {
        const isAllCurrentlySelected = (typesSelected ?? []).length === 0;
        const validTypes = selections.filter((s) => s !== optionAll && isType(s));

        if (
            (selections.includes(optionAll) && !isAllCurrentlySelected) ||
            validTypes.length === 0 ||
            validTypes.length === types.length
        ) {
            setTypesSelected(undefined);
            return;
        }

        setTypesSelected(validTypes);
    }

    const options = [
        <SelectOption key="All" value={optionAll}>
            All types
        </SelectOption>,
        <Divider key="Divider" />,
        ...types.map((type) => (
            <SelectOption key={type} value={type}>
                {getTypeText(type)}
            </SelectOption>
        )),
    ];

    return (
        <CheckboxSelect
            id="type-filter"
            selections={typesSelected ?? [optionAll]}
            onChange={onSelect}
            ariaLabel="Type filter menu items"
            toggleAriaLabel="Type filter menu toggle"
            placeholderText="Filter by type"
            isDisabled={isDisabled}
        >
            {options}
        </CheckboxSelect>
    );
}

export default SearchFilterTypes;
