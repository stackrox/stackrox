import { Fragment } from 'react';
import type { ReactElement } from 'react';
import {
    Divider,
    SelectGroup,
    SelectList,
    SelectOption,
    ToolbarItem,
} from '@patternfly/react-core';

import CheckboxSelect from 'Components/CheckboxSelect';
import type { SearchFilter } from 'types/search';
import { searchValueAsArray } from 'utils/searchUtils';

import type { OnSearchCallback, SelectSearchFilterAttribute } from '../types';

export type SearchFilterSelectInclusiveProps = {
    attribute: SelectSearchFilterAttribute;
    isSeparate?: boolean; // default false if within CompoundSearchFilter
    onSearch: OnSearchCallback;
    searchFilter: SearchFilter;
};

function SearchFilterSelectInclusive({
    attribute,
    isSeparate = false,
    onSearch,
    searchFilter,
}: SearchFilterSelectInclusiveProps): ReactElement {
    const { displayName, inputProps, searchTerm: category } = attribute;
    const selection = searchValueAsArray(searchFilter?.[category]);
    const toggleLabel = isSeparate ? displayName : `Filter by ${displayName}`;

    let content: JSX.Element | JSX.Element[] = (
        <SelectList>
            <SelectOption isDisabled>No options available</SelectOption>
        </SelectList>
    );

    if ('groupOptions' in inputProps && inputProps.groupOptions.length !== 0) {
        content = inputProps.groupOptions.map(({ name, options }, index) => {
            return (
                <Fragment key={name}>
                    <SelectGroup label={name}>
                        <SelectList>
                            {options.map((option) => (
                                <SelectOption
                                    key={option.value}
                                    hasCheckbox
                                    value={option.value}
                                    isSelected={selection.includes(option.value)}
                                >
                                    {option.label}
                                </SelectOption>
                            ))}
                        </SelectList>
                    </SelectGroup>
                    {index !== options.length - 1 && <Divider component="div" />}
                </Fragment>
            );
        });
    } else if ('options' in inputProps && inputProps.options.length !== 0) {
        content = (
            <SelectList>
                {inputProps.options.map((option) => (
                    <SelectOption
                        key={option.value}
                        hasCheckbox
                        value={option.value}
                        isSelected={selection.includes(option.value)}
                    >
                        {option.label}
                    </SelectOption>
                ))}
            </SelectList>
        );
    }

    return (
        <ToolbarItem>
            <CheckboxSelect
                selection={selection}
                onChange={(checked, _value) => {
                    onSearch([
                        {
                            action: checked ? 'SELECT_INCLUSIVE' : 'REMOVE',
                            category,
                            value: _value,
                        },
                    ]);
                }}
                ariaLabelMenu={`${toggleLabel} select menu`}
                toggleLabel={toggleLabel}
            >
                {content}
            </CheckboxSelect>
        </ToolbarItem>
    );
}

export default SearchFilterSelectInclusive;
