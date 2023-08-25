import { SearchInput } from '@patternfly/react-core';
import React, { ReactElement } from 'react';

type EntityNameSearchInputProps = {
    value: string;
    setValue: React.Dispatch<React.SetStateAction<string>>;
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
