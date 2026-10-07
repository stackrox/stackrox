import { useEffect, useState } from 'react';
import { SearchInput } from '@patternfly/react-core';

type SearchFilterNamespaceProps = {
    isDisabled: boolean;
    namespace: string | undefined;
    setNamespace: (namespace: string | undefined) => void;
};

function SearchFilterNamespace({
    isDisabled,
    namespace,
    setNamespace,
}: SearchFilterNamespaceProps) {
    const [inputValue, setInputValue] = useState(namespace ?? '');

    useEffect(() => setInputValue(namespace ?? ''), [namespace]);

    return (
        <SearchInput
            aria-label="Namespace filter"
            placeholder="Filter by namespace"
            value={inputValue}
            onChange={(_event, value) => setInputValue(value)}
            onSearch={() => setNamespace(inputValue || undefined)}
            onClear={() => {
                setInputValue('');
                setNamespace(undefined);
            }}
            isDisabled={isDisabled}
        />
    );
}

export default SearchFilterNamespace;
