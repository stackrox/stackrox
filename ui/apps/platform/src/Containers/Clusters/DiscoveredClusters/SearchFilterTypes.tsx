import React from 'react';
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
        const hadAllOption = (typesSelected ?? []).length === 0;
        const isSelectAll = selections.includes(optionAll) && !hadAllOption;
        const validTypes = selections.filter((s) => s !== optionAll && isType(s));

        if (isSelectAll || validTypes.length === 0) {
            setTypesSelected(undefined);
            return;
        }

        setTypesSelected(validTypes);
    }

    const options = types.map((type) => (
        <SelectOption key={type} value={type}>
            {getTypeText(type)}
        </SelectOption>
    ));
    options.push(
        <Divider key="Divider" />,
        <SelectOption key="All" value={optionAll}>
            All types
        </SelectOption>
    );

    return (
        <CheckboxSelect
            id="type-filter"
            selections={typesSelected ?? [optionAll]}
            onChange={onSelect}
            ariaLabel="Type filter menu items"
            placeholderText="Filter by type"
            isDisabled={isDisabled}
        >
            {options}
        </CheckboxSelect>
    );
}

export default SearchFilterTypes;
