import React, { useMemo, useState, ReactElement, useCallback } from 'react';
import {
    Button,
    Flex,
    FlexItem,
    Select,
    SelectOption,
    SelectList,
    MenuToggle,
    TextInputGroup,
    TextInputGroupMain,
    MenuFooter,
    Spinner,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import sortBy from 'lodash/sortBy';
import uniqBy from 'lodash/uniqBy';

import {
    Collection,
    CollectionSlim,
    getCollection,
    listCollections,
} from 'services/CollectionsService';
import { useCollectionFormSubmission } from 'Containers/Collections/hooks/useCollectionFormSubmission';
import { usePaginatedQuery } from 'hooks/usePaginatedQuery';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';
import { ReportScope } from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';

import CollectionsFormModal, {
    CollectionFormModalAction,
} from 'Containers/Collections/CollectionsFormModal';
import type { ClientCollection } from 'Containers/Collections/types';
import useRestQuery from 'hooks/useRestQuery';
import useAnalytics, { COLLECTION_CREATED } from 'hooks/useAnalytics';

const COLLECTION_PAGE_SIZE = 10;

type CollectionSelectionProps = {
    toggleId: string;
    id: string;
    selectedScope: ReportScope | null;
    onChange: (selection: CollectionSlim | null) => void;
    onBlur?: React.FocusEventHandler<HTMLDivElement>;
    onValidateField: (field: string) => void;
};

function CollectionSelection({
    toggleId,
    id,
    selectedScope,
    onChange,
    onBlur,
    onValidateField,
}: CollectionSelectionProps): ReactElement {
    const isRouteEnabled = useIsRouteEnabled();
    const isRouteEnabledForCollections = isRouteEnabled('collections');
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForCollections = hasReadWriteAccess('WorkflowAdministration');

    const { analyticsTrack } = useAnalytics();

    const [isOpen, setIsOpen] = useState(false);
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

    // If there is an existing collection selected, fetch the details for it. This allows
    // us to display the collection name in the dropdown even if the collection is not
    // in the current page of results.
    const selectedCollectionFetch = useCallback(() => {
        if (selectedScope?.id) {
            return getCollection(selectedScope?.id).request;
        }
        return Promise.resolve(undefined);
    }, [selectedScope?.id]);
    const { data: selectedScopeDetails } = useRestQuery(selectedCollectionFetch);

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

        // Add the individual pre-selected collection to the list of available scopes. If it already
        // exists in the list, it will be uniq'd out.
        if (selectedScopeDetails) {
            availableScopes.push(selectedScopeDetails.collection);
        }

        // This is inefficient due to the multiple loops and the fact that we are already tracking
        // uniqueness for the _server side_ values, but need to do it twice to handle possible client
        // side values. However, 'N' should be small here and we are memoizing the result.
        const sorted = sortBy(availableScopes, ({ name }) => name.toLowerCase());
        return uniqBy(sorted, 'id');
    }, [data, createdCollections, selectedScopeDetails]);

    function onOpenViewCollectionModal() {
        if (selectedScope) {
            setModalAction({ type: 'view', collectionId: selectedScope.id });
            setIsCollectionModalOpen((current) => !current);
        }
    }

    function onOpenCreateCollectionModal() {
        setIsOpen(false);
        setModalAction({ type: 'create' });
        setIsCollectionModalOpen((current) => !current);
    }

    function onScopeChange(
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        scopeId: string | number | undefined
    ) {
        const selectedCollection = sortedCollections.find(
            (collection) => collection.id === scopeId
        );
        if (selectedCollection) {
            setIsOpen(false);
            onChange(selectedCollection);
            setSearch('');
        }
    }

    function onToggleClick() {
        setIsOpen(!isOpen);
        if (isOpen) {
            setSearch('');
        }
    }

    function ensureOpen() {
        if (!isOpen) {
            setIsOpen(true);
        }
    }

    function onSearchChange(_event: React.FormEvent<HTMLInputElement>, value: string) {
        setSearch(value);
        onValidateField(id);
        ensureOpen();
    }

    function handleOpenChange(nextOpen: boolean) {
        setIsOpen(nextOpen);
        if (!nextOpen) {
            setSearch('');
        }
    }

    // Clears the search text when the user clicks away from the dropdown
    function handleBlur(event: React.FocusEvent<HTMLDivElement>) {
        setSearch('');
        onBlur?.(event);
    }

    // Loads the next page of collections when the user clicks "View more"
    function handleFetchNextPage() {
        fetchNextPage();
    }

    function handleCloseModal() {
        setIsCollectionModalOpen(false);
    }

    // Handles collection submission, updates local state, and tracks analytics
    function handleSubmitCollection(collection: ClientCollection) {
        return onSubmit(collection).then((collectionResponse) => {
            onChange(collectionResponse);
            setIsCollectionModalOpen(false);
            setCreatedCollections((oldCollections) => [...oldCollections, collectionResponse]);

            analyticsTrack({
                event: COLLECTION_CREATED,
                properties: { source: 'Vulnerability Reporting' },
            });
        });
    }

    const displayValue = useMemo(() => {
        if (!selectedScope?.id) {
            return '';
        }
        return (
            sortedCollections.find((collection) => collection.id === selectedScope.id)?.name || ''
        );
    }, [selectedScope?.id, sortedCollections]);

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            variant="typeahead"
            onClick={onToggleClick}
            isExpanded={isOpen}
            id={toggleId}
        >
            <TextInputGroup>
                <TextInputGroupMain
                    value={isOpen ? search : displayValue}
                    placeholder="Select a collection"
                    onChange={onSearchChange}
                    onFocus={ensureOpen}
                    onBlur={handleBlur}
                    autoComplete="off"
                    id={id}
                />
            </TextInputGroup>
        </MenuToggle>
    );

    const showLoadingSpinner = isFetchingNextPage;
    const showViewMoreButton = !isFetchingNextPage && !isEndOfResults;
    const showCreateCollectionFooter = hasWriteAccessForCollections && isRouteEnabledForCollections;

    return (
        <>
            <Flex
                direction={{ default: 'row' }}
                spaceItems={{ default: 'spaceItemsNone' }}
                alignItems={{ default: 'alignItemsFlexEnd' }}
            >
                <FlexItem>
                    <Select
                        id={`${id}-select`}
                        isOpen={isOpen}
                        selected={selectedScope?.id}
                        onSelect={onScopeChange}
                        onOpenChange={handleOpenChange}
                        toggle={toggle}
                        shouldFocusToggleOnSelect
                        popperProps={{
                            appendTo: () => document.body,
                            direction: 'up',
                        }}
                        onBlur={handleBlur}
                    >
                        <SelectList
                            style={{
                                maxHeight: '275px',
                                overflowY: 'auto',
                            }}
                        >
                            {sortedCollections.length === 0 && !showLoadingSpinner ? (
                                <SelectOption isDisabled>No results found</SelectOption>
                            ) : (
                                sortedCollections.map((collection) => (
                                    <SelectOption
                                        key={collection.id}
                                        value={collection.id}
                                        description={collection.description}
                                    >
                                        {collection.name}
                                    </SelectOption>
                                ))
                            )}
                            {showLoadingSpinner && (
                                <SelectOption isDisabled isAriaDisabled>
                                    <div className="pf-v5-u-text-align-center pf-v5-u-p-sm">
                                        <Spinner size="md" />
                                    </div>
                                </SelectOption>
                            )}
                            {showViewMoreButton && (
                                <SelectOption
                                    onClick={(e) => {
                                        e?.stopPropagation();
                                        handleFetchNextPage();
                                    }}
                                >
                                    <div className="pf-v5-u-text-align-center">
                                        <Button
                                            variant="link"
                                            isInline
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                handleFetchNextPage();
                                            }}
                                        >
                                            View more
                                        </Button>
                                    </div>
                                </SelectOption>
                            )}
                        </SelectList>
                        {showCreateCollectionFooter && (
                            <MenuFooter>
                                <Button
                                    variant="link"
                                    isInline
                                    onClick={onOpenCreateCollectionModal}
                                >
                                    Create collection
                                </Button>
                            </MenuFooter>
                        )}
                    </Select>
                </FlexItem>
                {isRouteEnabledForCollections && (
                    <FlexItem spacer={{ default: 'spacerMd' }}>
                        <Button
                            variant="tertiary"
                            onClick={onOpenViewCollectionModal}
                            isDisabled={!selectedScope}
                        >
                            View
                        </Button>
                    </FlexItem>
                )}
            </Flex>
            {isCollectionModalOpen && (
                <CollectionsFormModal
                    hasWriteAccessForCollections={hasWriteAccessForCollections}
                    modalAction={modalAction}
                    onClose={handleCloseModal}
                    configError={configError}
                    setConfigError={setConfigError}
                    onSubmit={handleSubmitCollection}
                />
            )}
        </>
    );
}

export default CollectionSelection;
