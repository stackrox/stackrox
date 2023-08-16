import React, { useMemo, useState, ReactElement, useCallback, useEffect } from 'react';
import {
    Button,
    ButtonVariant,
    Flex,
    FlexItem,
    Select,
    SelectOption,
    SelectProps,
    SelectVariant,
    ValidatedOptions,
} from '@patternfly/react-core';
import sortBy from 'lodash/sortBy';
import uniqBy from 'lodash/uniqBy';

import { Collection, CollectionSlim, listCollections } from 'services/CollectionsService';
import { useCollectionFormSubmission } from 'Containers/Collections/hooks/useCollectionFormSubmission';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { usePaginatedQuery } from 'hooks/usePaginatedQuery';
import { ReportScope } from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';

import CollectionsFormModal, {
    CollectionFormModalAction,
} from 'Containers/Collections/CollectionFormModal';

const COLLECTION_PAGE_SIZE = 10;

type CollectionSelectionProps = {
    toggleId: string;
    id: string;
    selectedScope: ReportScope | null;
    onChange: (selection: CollectionSlim | null) => void;
    allowCreate: boolean;
    onBlur?: React.FocusEventHandler<HTMLInputElement>;
    onValidateField: (field: string) => void;
};

function CollectionSelection({
    toggleId,
    id,
    selectedScope,
    onChange,
    allowCreate,
    onBlur,
    onValidateField,
}: CollectionSelectionProps): ReactElement {
    const { isOpen, onToggle } = useSelectToggle();
    const [modalAction, setModalAction] = useState<CollectionFormModalAction>({ type: 'create' });
    const [isCollectionModalOpen, setIsCollectionModalOpen] = useState(false);

    const { configError, setConfigError, onSubmit } = useCollectionFormSubmission(modalAction);
    const [search, setSearch] = useState('');

    const requestFn = useCallback(
        (page: number) => {
            return listCollections(
                { 'Collection Name': search },
                { field: 'Collection Name', reversed: false },
                page,
                COLLECTION_PAGE_SIZE
            ).request;
        },
        [search]
    );

    const { data, isEndOfResults, isFetchingNextPage, fetchNextPage } = usePaginatedQuery(
        requestFn,
        COLLECTION_PAGE_SIZE
    );

    // Combines the server-side fetched pages of collections data with the local cache
    // of created collections to create a flattened array sorted by name. This is intended to keep
    // the collection dropdown up to date with any collections that the user creates while in the form.
    //
    // Previously this was not needed since the component would refetch _all_ access scopes
    // upon creation of a new access scope, but we cannot do that efficiently since the collection dropdown
    // is paginated.
    //
    // This functionality can likely be removed if we move to a library based method of data fetching.
    const [createdCollections, setCreatedCollections] = useState<Collection[]>([]);
    const sortedCollections = useMemo(() => {
        const availableScopes: CollectionSlim[] = [...data.flat(), ...createdCollections];

        // This is inefficient due to the multiple loops and the fact that we are already tracking
        // uniqueness for the _server side_ values, but need to do it twice to handle possible client
        // side values. However, 'N' should be small here and we are memoizing the result.
        const sorted = sortBy(availableScopes, ({ name }) => name.toLowerCase());
        return uniqBy(sorted, 'id');
    }, [data, createdCollections]);

    // This makes sure that if a collection was deleted then we clear the scopeId
    useEffect(() => {
        if (!isFetchingNextPage) {
            const selectedCollection = sortedCollections.find(
                (collection) => collection.id === selectedScope?.id
            );
            if (!selectedCollection) {
                onChange(null);
            }
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data, isFetchingNextPage]);

    function onOpenViewCollectionModal() {
        if (selectedScope) {
            setModalAction({ type: 'view', collectionId: selectedScope.id });
            setIsCollectionModalOpen((current) => !current);
        }
    }

    function onOpenCreateCollectionModal() {
        setModalAction({ type: 'create' });
        setIsCollectionModalOpen((current) => !current);
    }

    function onScopeChange(_id, scopeId) {
        const selectedCollection = sortedCollections.find(
            (collection) => collection.id === scopeId
        );
        if (selectedCollection) {
            onToggle(false);
            onChange(selectedCollection);
        }
    }

    let selectLoadingVariant: SelectProps['loadingVariant'];

    if (isFetchingNextPage) {
        selectLoadingVariant = 'spinner';
    } else if (!isEndOfResults) {
        selectLoadingVariant = {
            text: 'View more',
            onClick: () => fetchNextPage(),
        };
    }

    return (
        <>
            <Flex
                direction={{ default: 'row' }}
                spaceItems={{ default: 'spaceItemsNone' }}
                alignItems={{ default: 'alignItemsFlexEnd' }}
                // Workaround to ensure there is enough space for the select menu when opened
                // at the bottom of a wizard step body (May no longer be needed after upgrade to PF5)
                className={isOpen ? 'pf-u-mb-3xl' : ''}
            >
                <FlexItem>
                    <Select
                        typeAheadAriaLabel={toggleId}
                        toggleId={toggleId}
                        id={id}
                        onSelect={onScopeChange}
                        selections={selectedScope?.id}
                        placeholderText="Select a collection"
                        variant={SelectVariant.typeahead}
                        isOpen={isOpen}
                        onToggle={onToggle}
                        onTypeaheadInputChanged={(value) => {
                            setSearch(value);
                            onValidateField(id);
                        }}
                        loadingVariant={selectLoadingVariant}
                        onBlur={(event) => {
                            setSearch('');
                            onBlur?.(event);
                        }}
                        style={{
                            maxHeight: '275px',
                            overflowY: 'auto',
                        }}
                        validated={ValidatedOptions.default}
                    >
                        {sortedCollections.map((collection) => (
                            <SelectOption
                                key={collection.id}
                                value={collection.id}
                                description={collection.description}
                            >
                                {collection.name}
                            </SelectOption>
                        ))}
                    </Select>
                </FlexItem>
                <FlexItem spacer={{ default: 'spacerMd' }}>
                    <Button
                        variant={ButtonVariant.tertiary}
                        onClick={onOpenViewCollectionModal}
                        isDisabled={!selectedScope}
                    >
                        View
                    </Button>
                </FlexItem>
                {allowCreate && (
                    <FlexItem>
                        <Button
                            variant={ButtonVariant.secondary}
                            onClick={onOpenCreateCollectionModal}
                        >
                            Create collection
                        </Button>
                    </FlexItem>
                )}
            </Flex>
            {isCollectionModalOpen && (
                <CollectionsFormModal
                    hasWriteAccessForCollections={allowCreate}
                    modalAction={modalAction}
                    onClose={() => setIsCollectionModalOpen(false)}
                    configError={configError}
                    setConfigError={setConfigError}
                    onSubmit={(collection) =>
                        onSubmit(collection).then((collectionResponse) => {
                            onChange(collectionResponse);
                            setIsCollectionModalOpen(false);
                            setCreatedCollections((oldCollections) => [
                                ...oldCollections,
                                collectionResponse,
                            ]);
                        })
                    }
                />
            )}
        </>
    );
}

export default CollectionSelection;
