import React, { ReactElement, useCallback, useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    Breadcrumb,
    BreadcrumbItem,
    Button,
    Divider,
    Dropdown,
    DropdownItem,
    DropdownSeparator,
    DropdownToggle,
    Flex,
    FlexItem,
    Title,
} from '@patternfly/react-core';
import { useMediaQuery } from 'react-responsive';

import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import {
    CollectionResponse,
    createCollection,
    deleteCollection,
    getCollection,
    listCollections,
    ResolvedCollectionResponse,
    updateCollection,
} from 'services/CollectionsService';
import { CaretDownIcon } from '@patternfly/react-icons';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { collectionsBasePath } from 'routePaths';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import useToasts from 'hooks/patternfly/useToasts';
import { values } from 'lodash';
import { CollectionPageAction } from './collections.utils';
import CollectionForm from './CollectionForm';
import { generateRequest } from './converter';
import { Collection } from './types';

export type CollectionsFormPageProps = {
    hasWriteAccessForCollections: boolean;
    pageAction: CollectionPageAction;
};

const defaultCollectionData: Omit<CollectionResponse, 'id'> = {
    name: '',
    description: '',
    inUse: false,
    embeddedCollections: [],
    resourceSelectors: [],
};

const noopRequest = {
    request: Promise.resolve<{
        collection: Omit<CollectionResponse, 'id'>;
        embeddedCollections: CollectionResponse[];
    }>({ collection: defaultCollectionData, embeddedCollections: [] }),
    cancel: () => {},
};

function getEmbeddedCollections({ collection }: ResolvedCollectionResponse): Promise<{
    collection: CollectionResponse;
    embeddedCollections: CollectionResponse[];
}> {
    if (collection.embeddedCollections.length === 0) {
        return Promise.resolve({ collection, embeddedCollections: [] });
    }
    const idSearchString = collection.embeddedCollections.map(({ id }) => id).join(',');
    const searchFilter = { 'Collection ID': idSearchString };
    const { request } = listCollections(searchFilter, { field: 'name', reversed: false });
    return request.then((embeddedCollections) => ({ collection, embeddedCollections }));
}

function CollectionsFormPage({
    hasWriteAccessForCollections,
    pageAction,
}: CollectionsFormPageProps) {
    const history = useHistory();
    const isLargeScreen = useMediaQuery({ query: '(min-width: 992px)' }); // --pf-global--breakpoint--lg
    const collectionId = pageAction.type !== 'create' ? pageAction.collectionId : undefined;
    const collectionFetcher = useCallback(() => {
        if (!collectionId) {
            return noopRequest;
        }
        const { request, cancel } = getCollection(collectionId);
        return { request: request.then(getEmbeddedCollections), cancel };
    }, [collectionId]);
    const { data, loading, error } = useRestQuery(collectionFetcher);

    const { toasts, addToast, removeToast } = useToasts();

    const [isDeleting, setIsDeleting] = useState(false);
    const [deleteId, setDeleteId] = useState<string | null>(null);

    const {
        isOpen: menuIsOpen,
        toggleSelect: toggleMenu,
        closeSelect: closeMenu,
    } = useSelectToggle();

    const {
        isOpen: isDrawerOpen,
        toggleSelect: toggleDrawer,
        closeSelect: closeDrawer,
        openSelect: openDrawer,
    } = useSelectToggle(isLargeScreen);

    function onEditCollection(id: string) {
        history.push({
            pathname: `${collectionsBasePath}/${id}`,
            search: 'action=edit',
        });
    }

    function onCloneCollection(id: string) {
        history.push({
            pathname: `${collectionsBasePath}/${id}`,
            search: 'action=clone',
        });
    }

    function onConfirmDeleteCollection() {
        if (!deleteId) {
            return;
        }
        setIsDeleting(true);
        deleteCollection(deleteId)
            .request.then(history.goBack)
            .catch((err) => {
                addToast(
                    `Could not delete collection ${data?.collection?.name ?? ''}`,
                    'danger',
                    err.message
                );
            })
            .finally(() => {
                setDeleteId(null);
                setIsDeleting(false);
            });
    }

    function onCancelDeleteCollection() {
        setDeleteId(null);
    }

    function onSubmit(collection: Collection): Promise<void> {
        if (pageAction.type === 'view') {
            // Logically should not happen, but just in case
            return Promise.reject();
        }

        const saveServiceCall =
            pageAction.type === 'edit'
                ? (payload) => updateCollection(pageAction.collectionId, payload)
                : (payload) => createCollection(payload);

        const requestPayload = generateRequest(collection);
        const { request } = saveServiceCall(requestPayload);

        return request
            .then(() => {
                history.push({ pathname: `${collectionsBasePath}` });
            })
            .catch((err) => {
                addToast(
                    `There was an error saving collection '${values.name}'`,
                    'danger',
                    err.message
                );
            });
    }

    let content: ReactElement | undefined;

    if (error) {
        content = (
            <>
                {error.message}
                {/* TODO - Handle UI for network errors */}
            </>
        );
    } else if (loading) {
        content = <>{/* TODO - Handle UI for loading state */}</>;
    } else if (data) {
        const pageTitle = pageAction.type === 'create' ? 'Create collection' : data.collection.name;
        content = (
            <CollectionForm
                hasWriteAccessForCollections={hasWriteAccessForCollections}
                action={pageAction}
                collectionData={data}
                useInlineDrawer={isLargeScreen}
                isDrawerOpen={isDrawerOpen}
                toggleDrawer={toggleDrawer}
                onSubmit={onSubmit}
                appendTableLinkAction={() => {
                    /* TODO */
                }}
                headerContent={
                    <>
                        <Breadcrumb className="pf-u-my-xs pf-u-px-lg pf-u-py-md">
                            <BreadcrumbItemLink to={collectionsBasePath}>
                                Collections
                            </BreadcrumbItemLink>
                            <BreadcrumbItem>{pageTitle}</BreadcrumbItem>
                        </Breadcrumb>
                        <Divider component="div" />
                        <Flex
                            className="pf-u-p-lg"
                            direction={{ default: 'column', md: 'row' }}
                            alignItems={{ default: 'alignItemsFlexStart', md: 'alignItemsCenter' }}
                        >
                            <Title className="pf-u-flex-grow-1" headingLevel="h1">
                                {pageTitle}
                            </Title>
                            <FlexItem align={{ default: 'alignLeft', md: 'alignRight' }}>
                                {pageAction.type === 'view' && hasWriteAccessForCollections && (
                                    <>
                                        <Dropdown
                                            onSelect={closeMenu}
                                            toggle={
                                                <DropdownToggle
                                                    isPrimary
                                                    onToggle={toggleMenu}
                                                    toggleIndicator={CaretDownIcon}
                                                >
                                                    Actions
                                                </DropdownToggle>
                                            }
                                            isOpen={menuIsOpen}
                                            dropdownItems={[
                                                <DropdownItem
                                                    key="Edit collection"
                                                    component="button"
                                                    onClick={() =>
                                                        onEditCollection(pageAction.collectionId)
                                                    }
                                                >
                                                    Edit collection
                                                </DropdownItem>,
                                                <DropdownItem
                                                    key="Clone collection"
                                                    component="button"
                                                    onClick={() =>
                                                        onCloneCollection(pageAction.collectionId)
                                                    }
                                                >
                                                    Clone collection
                                                </DropdownItem>,
                                                <DropdownSeparator key="Separator" />,
                                                <DropdownItem
                                                    key="Delete collection"
                                                    component="button"
                                                    isDisabled={data.collection.inUse}
                                                    onClick={() =>
                                                        setDeleteId(pageAction.collectionId)
                                                    }
                                                >
                                                    {data.collection.inUse
                                                        ? 'Cannot delete (in use)'
                                                        : 'Delete collection'}
                                                </DropdownItem>,
                                            ]}
                                        />
                                        <Divider
                                            className="pf-u-px-xs"
                                            orientation={{ default: 'vertical' }}
                                        />
                                    </>
                                )}
                                {isDrawerOpen ? (
                                    <Button variant="secondary" onClick={closeDrawer}>
                                        Hide collection results
                                    </Button>
                                ) : (
                                    <Button variant="secondary" onClick={openDrawer}>
                                        Preview collection results
                                    </Button>
                                )}
                            </FlexItem>
                        </Flex>
                        <Divider component="div" />
                    </>
                }
            />
        );
    }

    return (
        <>
            {content}
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }) => (
                    <Alert
                        key={key}
                        variant={variant}
                        title={title}
                        timeout
                        onTimeout={() => removeToast(key)}
                        actionClose={
                            <AlertActionCloseButton
                                title={title}
                                variantLabel={variant}
                                onClose={() => removeToast(key)}
                            />
                        }
                    >
                        {children}
                    </Alert>
                ))}
            </AlertGroup>
            <ConfirmationModal
                ariaLabel="Confirm delete"
                confirmText="Delete"
                isLoading={isDeleting}
                isOpen={deleteId !== null}
                onConfirm={onConfirmDeleteCollection}
                onCancel={onCancelDeleteCollection}
            >
                Are you sure you want to delete this collection?
            </ConfirmationModal>
        </>
    );
}

export default CollectionsFormPage;
