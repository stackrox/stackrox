import React, { useMemo, useState, ReactElement, useCallback } from 'react';
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

import {
    Collection,
    CollectionSlim,
    getCollection,
    listCollections,
} from 'services/CollectionsService';
import { useCollectionFormSubmission } from 'Containers/Collections/hooks/useCollectionFormSubmission';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { usePaginatedQuery } from 'hooks/usePaginatedQuery';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';
import { ReportScope } from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';

import CollectionsFormModal, {
    CollectionFormModalAction,
} from 'Containers/Collections/CollectionFormModal';
import useRestQuery from 'hooks/useRestQuery';
import useAnalytics, { COLLECTION_CREATED } from 'hooks/useAnalytics';

const COLLECTION_PAGE_SIZE = 10;

type CollectionSelectionProps = {
    toggleId: string;
    id: string;
    selectedScope: ReportScope | null;
    onChange: (selection: CollectionSlim | null) => void;
    onBlur?: React.FocusEventHandler<HTMLInputElement>;
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
        onToggle(false);
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
                        footer={
                            hasWriteAccessForCollections &&
                            isRouteEnabledForCollections && (
                                <Button
                                    variant="link"
                                    isInline
                                    onClick={onOpenCreateCollectionModal}
                                >
                                    Create collection
                                </Button>
                            )
                        }
                        menuAppendTo={() => document.body}
                        direction="up"
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
                {isRouteEnabledForCollections && (
                    <FlexItem spacer={{ default: 'spacerMd' }}>
                        <Button
                            variant={ButtonVariant.tertiary}
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

                            analyticsTrack({
                                event: COLLECTION_CREATED,
                                properties: { source: 'Vulnerability Reporting' },
                            });
                        })
                    }
                />
            )}
        </>
    );
}

export default CollectionSelection;
