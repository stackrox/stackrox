import React, { ReactElement, useState } from 'react';
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
    PageSection,
    Title,
} from '@patternfly/react-core';
import { useMediaQuery } from 'react-responsive';

import { createCollection, deleteCollection, updateCollection } from 'services/CollectionsService';
import { CaretDownIcon } from '@patternfly/react-icons';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { collectionsBasePath } from 'routePaths';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import useToasts from 'hooks/patternfly/useToasts';
import { values } from 'lodash';
import { CollectionPageAction } from './collections.utils';
import CollectionFormDrawer from './CollectionFormDrawer';
import { generateRequest } from './converter';
import { Collection } from './types';
import useCollection from './hooks/useCollection';
import CollectionsFormModal from './CollectionFormModal';

export type CollectionsFormPageProps = {
    hasWriteAccessForCollections: boolean;
    pageAction: CollectionPageAction;
};

function CollectionsFormPage({
    hasWriteAccessForCollections,
    pageAction,
}: CollectionsFormPageProps) {
    const history = useHistory();
    const isLargeScreen = useMediaQuery({ query: '(min-width: 992px)' }); // --pf-global--breakpoint--lg
    const collectionId = pageAction.type !== 'create' ? pageAction.collectionId : undefined;

    const { data, loading, error } = useCollection(collectionId);

    const { toasts, addToast, removeToast } = useToasts();

    const [isDeleting, setIsDeleting] = useState(false);
    const [deleteId, setDeleteId] = useState<string | null>(null);
    const [modalCollectionId, setModalCollectionId] = useState<string | null>(null);

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
            <CollectionFormDrawer
                hasWriteAccessForCollections={hasWriteAccessForCollections}
                action={pageAction}
                collectionData={data}
                isInlineDrawer={isLargeScreen}
                isDrawerOpen={isDrawerOpen}
                toggleDrawer={toggleDrawer}
                onSubmit={onSubmit}
                appendTableLinkAction={setModalCollectionId}
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
                                        Hide results
                                    </Button>
                                ) : (
                                    <Button variant="secondary" onClick={openDrawer}>
                                        Preview results
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
        <PageSection className="pf-u-h-100" padding={{ default: 'noPadding' }}>
            {content}
            {modalCollectionId && (
                <CollectionsFormModal
                    hasWriteAccessForCollections={hasWriteAccessForCollections}
                    collectionId={modalCollectionId}
                    onClose={() => setModalCollectionId(null)}
                />
            )}
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
        </PageSection>
    );
}

export default CollectionsFormPage;
