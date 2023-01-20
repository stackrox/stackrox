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
} from '@patternfly/react-core';
import sortBy from 'lodash/sortBy';
import uniqBy from 'lodash/uniqBy';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import { Collection, listCollections } from 'services/CollectionsService';
import CollectionsFormModal from 'Containers/Collections/CollectionFormModal';
import { useCollectionFormSubmission } from 'Containers/Collections/hooks/useCollectionFormSubmission';
import { useFormik } from 'formik';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { usePaginatedQuery } from 'hooks/usePaginatedQuery';

const COLLECTION_PAGE_SIZE = 10;

type CollectionSelectionProps = {
    scopeId: string;
    setFieldValue: ReturnType<typeof useFormik<{ scopeId: string }>>['setFieldValue'];
    allowCreate: boolean;
};

function CollectionSelection({
    scopeId,
    setFieldValue,
    allowCreate,
}: CollectionSelectionProps): ReactElement {
    const { isOpen, onToggle } = useSelectToggle();
    const { configError, setConfigError, onSubmit } = useCollectionFormSubmission({
        type: 'create',
    });
    const [search, setSearch] = useState('');

    const [isCollectionModalOpen, setIsCollectionModalOpen] = useState(false);

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
        // This is inefficient due to the multiple loops and the fact that we are already tracking
        // uniqueness for the _server side_ values, but need to do it twice to handle possible client
        // side values. However, 'N' should be small here and we are memoizing the result.
        const sorted = sortBy(data.flat().concat(createdCollections), ({ name }) =>
            name.toLowerCase()
        );
        return uniqBy(sorted, 'id');
    }, [data, createdCollections]);

    function onToggleCollectionModal() {
        setIsCollectionModalOpen((current) => !current);
    }

    function onScopeChange(_id, selection) {
        onToggle(false);
        return setFieldValue('scopeId', selection);
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
            <Flex alignItems={{ default: 'alignItemsFlexEnd' }}>
                <FlexItem>
                    <FormLabelGroup
                        className="pf-u-mb-md"
                        isRequired
                        label="Configure report scope"
                        fieldId="scopeId"
                        touched={{}}
                        errors={{}}
                    >
                        <Select
                            id="scopeId"
                            onSelect={onScopeChange}
                            selections={scopeId}
                            placeholderText="Select a collection"
                            variant={SelectVariant.typeahead}
                            isOpen={isOpen}
                            onToggle={onToggle}
                            onTypeaheadInputChanged={setSearch}
                            loadingVariant={selectLoadingVariant}
                            isInputValuePersisted
                        >
                            {sortedCollections.map(({ id, name, description }) => (
                                <SelectOption key={id} value={id} description={description}>
                                    {name}
                                </SelectOption>
                            ))}
                        </Select>
                    </FormLabelGroup>
                </FlexItem>
                {allowCreate && (
                    <FlexItem>
                        <Button
                            className="pf-u-mb-md"
                            variant={ButtonVariant.secondary}
                            onClick={onToggleCollectionModal}
                        >
                            Create collection
                        </Button>
                    </FlexItem>
                )}
            </Flex>
            {isCollectionModalOpen && (
                <CollectionsFormModal
                    hasWriteAccessForCollections={allowCreate}
                    modalAction={{ type: 'create' }}
                    onClose={() => setIsCollectionModalOpen(false)}
                    configError={configError}
                    setConfigError={setConfigError}
                    onSubmit={(collection) =>
                        onSubmit(collection).then((collectionResponse) =>
                            setFieldValue('scopeId', collectionResponse.id).then(() => {
                                setIsCollectionModalOpen(false);
                                setCreatedCollections((oldCollections) => [
                                    ...oldCollections,
                                    collectionResponse,
                                ]);
                            })
                        )
                    }
                />
            )}
        </>
    );
}

export default CollectionSelection;
