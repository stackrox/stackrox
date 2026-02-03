import { useState } from 'react';
import type { ReactElement } from 'react';
import { SearchInput, ToolbarItem } from '@patternfly/react-core';

import type { GenericSearchFilterAttribute, OnSearchCallback } from '../types';

export type SearchFilterTextProps = {
    attribute: GenericSearchFilterAttribute;
    onSearch: OnSearchCallback;
    // does not depend on searchFilter
};

function SearchFilterText({ attribute, onSearch }: SearchFilterTextProps): ReactElement {
    const { filterChipLabel, searchTerm: category } = attribute;
    const textLabel = `Filter results by ${filterChipLabel}`;

    const [value, setValue] = useState('');

    return (
        <ToolbarItem className="pf-v6-u-flex-grow-1">
            <SearchInput
                aria-label={textLabel}
                placeholder={textLabel}
                value={value}
                onChange={(_event, _value) => setValue(_value)}
                onSearch={(_event, _value) => {
                    onSearch([
                        {
                            action: 'APPEND',
                            category,
                            value: _value,
                        },
                    ]);
                    setValue('');
                }}
                onClear={() => setValue('')}
                submitSearchButtonLabel="Apply text input to search"
            />
        </ToolbarItem>
    );
}

export default SearchFilterText;
