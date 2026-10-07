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
    return (
        <SearchInput
            aria-label="Namespace filter"
            placeholder="Filter by namespace"
            value={namespace ?? ''}
            onChange={(_event, value) => setNamespace(value || undefined)}
            onClear={() => setNamespace(undefined)}
            isDisabled={isDisabled}
        />
    );
}

export default SearchFilterNamespace;
