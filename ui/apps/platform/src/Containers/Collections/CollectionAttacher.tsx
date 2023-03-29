import React, { useState } from 'react';
import { Alert, Button, Flex, SearchInput } from '@patternfly/react-core';

import BacklogListSelector, {
    BacklogListSelectorProps,
} from 'Components/PatternFly/BacklogListSelector';
import { Collection } from 'services/CollectionsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useEmbeddedCollections from './hooks/useEmbeddedCollections';

// We need to use `startsWith` instead of `includes` here, since search values sent to the API use
// a prefix match. If we filter by substring, collections matching the substring will appear in
// the "attached" list, but not in the "available" list, since the former are cached client side.
function compareNameLowercase(search: string, item: { name: string }): boolean {
    return item.name.toLowerCase().startsWith(search.toLowerCase());
}

export type CollectionAttacherProps = {
    // A collection ID that should not be visible in the collection attacher component. This is
    // used when editing a collection to prevent reference cycles.
    excludedCollectionId: string | null;
    initialEmbeddedCollections: Collection[];
    onSelectionChange: (collections: Collection[]) => void;
    collectionTableCells: BacklogListSelectorProps<Collection>['cells'];
};

function CollectionAttacher({
    excludedCollectionId,
    initialEmbeddedCollections,
    onSelectionChange,
    collectionTableCells,
}: CollectionAttacherProps) {
    const [searchInput, setSearchInput] = useState('');
    const [searchValue, setSearchValue] = useState('');
    const embedded = useEmbeddedCollections(excludedCollectionId, initialEmbeddedCollections);
    const { attached, detached, attach, detach, hasMore, fetchMore, onSearch } = embedded;
    const { isFetchingMore, fetchMoreError } = embedded;

    function onSearchInputChange(_event, value) {
        setSearchInput(value);
    }

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXl' }}>
            <SearchInput
                aria-label="Filter by name"
                placeholder="Filter by name"
                value={searchInput}
                onChange={onSearchInputChange}
                onSearch={() => {
                    onSearch(searchInput);
                    setSearchValue(searchInput);
                }}
                onClear={() => {
                    onSearch('');
                    setSearchValue('');
                    setSearchInput('');
                }}
            />
            <BacklogListSelector
                selectedOptions={attached}
                deselectedOptions={detached}
                onSelectItem={({ id }) => attach(id)}
                onDeselectItem={({ id }) => detach(id)}
                onSelectionChange={onSelectionChange}
                rowKey={({ id }) => id}
                cells={collectionTableCells}
                selectedLabel="Attached collections"
                deselectedLabel="Available collections"
                selectButtonText="Attach"
                deselectButtonText="Detach"
                searchFilter={(item) => compareNameLowercase(searchValue, item)}
            />
            {fetchMoreError && (
                <Alert
                    variant="danger"
                    isInline
                    title="There was an error loading more collections"
                >
                    {getAxiosErrorMessage(fetchMoreError)}
                </Alert>
            )}
            {hasMore && (
                <Button
                    className="pf-u-align-self-flex-start"
                    variant="secondary"
                    onClick={() => fetchMore(searchValue)}
                    isLoading={isFetchingMore}
                >
                    View more
                </Button>
            )}
        </Flex>
    );
}

export default CollectionAttacher;
