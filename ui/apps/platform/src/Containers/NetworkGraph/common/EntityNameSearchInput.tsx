import React from 'react';
import type { Dispatch, ReactElement, SetStateAction } from 'react';
import { SearchInput } from '@patternfly/react-core';

type EntityNameSearchInputProps = {
    value: string;
    setValue: Dispatch<SetStateAction<string>>;
};

function EntityNameSearchInput({ value, setValue }: EntityNameSearchInputProps): ReactElement {
    function onSearchInputChange(_event, newValue) {
        setValue(newValue);
    }

    return (
        <SearchInput
            placeholder="Filter by entity name"
            value={value}
            onChange={onSearchInputChange}
            onClear={() => setValue('')}
        />
    );
}

export default EntityNameSearchInput;
