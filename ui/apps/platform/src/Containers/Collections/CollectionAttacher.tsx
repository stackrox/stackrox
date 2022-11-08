import React, { useMemo, useState } from 'react';
import { Alert, Button, debounce, Flex, SearchInput, Truncate } from '@patternfly/react-core';

import BacklogListSelector from 'Components/PatternFly/BacklogListSelector';
import { CollectionResponse } from 'services/CollectionsService';
import useEmbeddedCollections from './hooks/useEmbeddedCollections';

export type CollectionAttacherProps = {
    // A collection ID that should not be visible in the collection attacher component. This is
    // used when editing a collection to prevent reference cycles.
    excludedCollectionId: string | null;
    initialEmbeddedCollections: CollectionResponse[];
    onSelectionChange: (collections: CollectionResponse[]) => void;
    onItemClick: (collectionId: string) => void;
};

function compareNameLowercase(search: string): (item: { name: string }) => boolean {
    return ({ name }) => name.toLowerCase().includes(search.toLowerCase());
}

function CollectionAttacher({
    excludedCollectionId,
    initialEmbeddedCollections,
    onSelectionChange,
    onItemClick,
}: CollectionAttacherProps) {
    const [search, setSearch] = useState('');
    const embedded = useEmbeddedCollections(excludedCollectionId, initialEmbeddedCollections);
    const { attached, detached, attach, detach, hasMore, fetchMore, onSearch } = embedded;
    const { isFetchingMore, fetchMoreError } = embedded;

    const onSearchInputChange = useMemo(
        () =>
            debounce((value: string) => {
                setSearch(value);
                onSearch(value);
            }, 800),
        [onSearch]
    );

    const selectorListCells = [
        {
            name: 'Name',
            render: ({ id, name }) => (
                <Button
                    variant="link"
                    className="pf-u-pl-0"
                    isInline
                    onClick={() => onItemClick(id)}
                >
                    {name}
                </Button>
            ),
        },
        {
            name: 'Description',
            render: ({ description }) => <Truncate content={description} />,
        },
    ];

    const selectedOptions = attached.filter(compareNameLowercase(search));
    const deselectedOptions = detached.filter(compareNameLowercase(search));

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXl' }}>
            <SearchInput
                aria-label="Filter by name"
                placeholder="Filter by name"
                value={search}
                onChange={onSearchInputChange}
            />
            <BacklogListSelector
                selectedOptions={selectedOptions}
                deselectedOptions={deselectedOptions}
                onSelectItem={({ id }) => attach(id)}
                onDeselectItem={({ id }) => detach(id)}
                onSelectionChange={onSelectionChange}
                rowKey={({ id }) => id}
                cells={selectorListCells}
                selectedLabel="Attached collections"
                deselectedLabel="Detached collections"
                selectButtonText="Attach"
                deselectButtonText="Detach"
            />
            {fetchMoreError && (
                <Alert
                    variant="danger"
                    isInline
                    title="There was an error loading more collections"
                />
            )}
            {hasMore && (
                <Button
                    className="pf-u-align-self-flex-start"
                    variant="secondary"
                    onClick={() => fetchMore(search)}
                    isLoading={isFetchingMore}
                >
                    View more
                </Button>
            )}
        </Flex>
    );
}

export default CollectionAttacher;
