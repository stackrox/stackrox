import React, { ReactElement, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Button,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Title,
    Tooltip,
    Truncate,
} from '@patternfly/react-core';
import {
    Dropdown,
    DropdownItem,
    DropdownSeparator,
    DropdownToggle,
} from '@patternfly/react-core/deprecated';
import { useMediaQuery } from 'react-responsive';

import { deleteCollection } from 'services/CollectionsService';
import { CaretDownIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { collectionsBasePath } from 'routePaths';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import useToasts from 'hooks/patternfly/useToasts';
import PageTitle from 'Components/PageTitle';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useAnalytics, { COLLECTION_CREATED } from 'hooks/useAnalytics';
import { CollectionPageAction } from './collections.utils';
import CollectionFormDrawer, { CollectionFormDrawerProps } from './CollectionFormDrawer';
import useCollection from './hooks/useCollection';
import CollectionsFormModal from './CollectionFormModal';
import CollectionLoadError from './CollectionLoadError';
import { useCollectionFormSubmission } from './hooks/useCollectionFormSubmission';

export type CollectionsFormPageProps = {
    hasWriteAccessForCollections: boolean;
    pageAction: CollectionPageAction;
};

function getPageTitle(
    pageAction: CollectionPageAction,
    pageData: ReturnType<typeof useCollection>['data']
): string {
    const pageTitleSuffix = pageData ? ` - ${pageData.collection.name}` : '';
    const titles = {
        create: `Create Collection`,
        clone: `Clone Collection${pageTitleSuffix}`,
        edit: `Edit Collection${pageTitleSuffix}`,
        view: `Collection${pageTitleSuffix}`,
    };
    return titles[pageAction.type];
}

function CollectionsFormPage({
    hasWriteAccessForCollections,
    pageAction,
}: CollectionsFormPageProps) {
    const navigate = useNavigate();
    const isXLargeScreen = useMediaQuery({ query: '(min-width: 1200px)' }); // --pf-v5-global--breakpoint--xl
    const collectionId = pageAction.type !== 'create' ? pageAction.collectionId : undefined;

    const { analyticsTrack } = useAnalytics();

    const { data, isLoading, error } = useCollection(collectionId);

    const { toasts, addToast, removeToast } = useToasts();

    const [isDeleting, setIsDeleting] = useState(false);
    const [deleteId, setDeleteId] = useState<string | null>(null);
    const [modalCollectionId, setModalCollectionId] = useState<string | null>(null);

    const { configError, setConfigError, onSubmit } = useCollectionFormSubmission(pageAction);
    const configErrorAlertElem = useRef<HTMLDivElement | null>(null);

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
    } = useSelectToggle(isXLargeScreen);

    function onEditCollection(id: string) {
        navigate(`${collectionsBasePath}/${id}?action=edit`);
    }

    function onCloneCollection(id: string) {
        navigate(`${collectionsBasePath}/${id}?action=clone`);
    }

    function onConfirmDeleteCollection() {
        if (!deleteId) {
            return;
        }
        setIsDeleting(true);
        deleteCollection(deleteId)
            .request.then(() => navigate(-1))
            .catch((err) => {
                const message = getAxiosErrorMessage(err);
                addToast(
                    `Could not delete collection ${data?.collection?.name ?? ''}`,
                    'danger',
                    message
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

    function scrollToTop() {
        const scrollTargetElem = configErrorAlertElem.current;
        if (scrollTargetElem) {
            scrollTargetElem.scrollIntoView({ behavior: 'smooth' });
        }
    }

    function getCollectionTableCells(
        collectionErrorId: string | undefined
    ): ReturnType<CollectionFormDrawerProps['getCollectionTableCells']> {
        return [
            {
                name: 'Name',
                render: ({ id, name }) => (
                    <Flex
                        alignItems={{ default: 'alignItemsCenter' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                        flexWrap={{ default: 'nowrap' }}
                    >
                        <Button
                            variant="link"
                            isInline
                            onClick={() => setModalCollectionId(id)}
                            isDanger={collectionErrorId === id}
                        >
                            {name}
                        </Button>
                        {collectionErrorId === id ? (
                            <Tooltip content="This collection forms a loop with its parent and cannot be attached">
                                <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--100)" />
                            </Tooltip>
                        ) : null}
                    </Flex>
                ),
            },
            {
                name: 'Description',
                render: ({ description }) => <Truncate content={description} />,
            },
        ];
    }

    let content: ReactElement | undefined;

    if (error) {
        content = (
            <>
                <Breadcrumb className="pf-v5-u-my-xs pf-v5-u-px-lg pf-v5-u-py-md">
                    <BreadcrumbItemLink to={collectionsBasePath}>Collections</BreadcrumbItemLink>
                </Breadcrumb>
                <Divider component="div" />
                <CollectionLoadError
                    title="There was an error loading this collection"
                    error={error}
                />
            </>
        );
    } else if (isLoading) {
        content = (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    } else if (data) {
        const pageName = pageAction.type === 'create' ? 'Create collection' : data.collection.name;
        content = (
            <CollectionFormDrawer
                hasWriteAccessForCollections={hasWriteAccessForCollections}
                action={pageAction}
                collectionData={data}
                isInlineDrawer={isXLargeScreen}
                isDrawerOpen={isDrawerOpen}
                toggleDrawer={toggleDrawer}
                onSubmit={(collection) =>
                    onSubmit(collection)
                        .then(() => {
                            navigate(`${collectionsBasePath}`);
                            if (pageAction.type === 'create') {
                                analyticsTrack({
                                    event: COLLECTION_CREATED,
                                    properties: { source: 'Collections' },
                                });
                            }
                        })
                        .catch((err) => {
                            scrollToTop();
                            return Promise.reject(err);
                        })
                }
                onCancel={() => {
                    navigate(`${collectionsBasePath}`);
                }}
                configError={configError}
                setConfigError={setConfigError}
                getCollectionTableCells={getCollectionTableCells}
                headerContent={
                    <>
                        <Breadcrumb className="pf-v5-u-my-xs pf-v5-u-px-lg pf-v5-u-py-md">
                            <BreadcrumbItemLink to={collectionsBasePath}>
                                Collections
                            </BreadcrumbItemLink>
                            <BreadcrumbItem>{pageName}</BreadcrumbItem>
                        </Breadcrumb>
                        <Divider component="div" />
                        <Flex
                            className="pf-v5-u-p-lg"
                            direction={{ default: 'column', md: 'row' }}
                            alignItems={{ default: 'alignItemsFlexStart', md: 'alignItemsCenter' }}
                        >
                            <Title className="pf-v5-u-flex-grow-1" headingLevel="h1">
                                {pageName}
                            </Title>
                            <FlexItem align={{ default: 'alignLeft', md: 'alignRight' }}>
                                {pageAction.type === 'view' && hasWriteAccessForCollections && (
                                    <>
                                        <Dropdown
                                            onSelect={closeMenu}
                                            toggle={
                                                <DropdownToggle
                                                    toggleVariant="primary"
                                                    onToggle={(_e, v) => toggleMenu(v)}
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
                                                    onClick={() =>
                                                        setDeleteId(pageAction.collectionId)
                                                    }
                                                >
                                                    Delete collection
                                                </DropdownItem>,
                                            ]}
                                        />
                                        <Divider
                                            className="pf-v5-u-px-xs"
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
                        {/* This <div> gives us a reliable `ref` to use as a scroll target when an error occurs */}
                        <div ref={configErrorAlertElem}>
                            {configError && (
                                <Alert
                                    className="pf-v5-u-m-md"
                                    title={configError.message}
                                    component="p"
                                    variant="danger"
                                    isInline
                                >
                                    {configError?.type === 'UnknownError'
                                        ? configError.details
                                        : ''}
                                </Alert>
                            )}
                        </div>
                        <Divider component="div" />
                    </>
                }
            />
        );
    }

    return (
        <PageSection className="pf-v5-u-h-100" padding={{ default: 'noPadding' }}>
            <PageTitle title={getPageTitle(pageAction, data)} />
            {content}
            {modalCollectionId && (
                <CollectionsFormModal
                    hasWriteAccessForCollections={hasWriteAccessForCollections}
                    modalAction={{
                        type: 'view',
                        collectionId: modalCollectionId,
                    }}
                    onClose={() => setModalCollectionId(null)}
                />
            )}
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }) => (
                    <Alert
                        key={key}
                        variant={variant}
                        title={title}
                        component="p"
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
                confirmText="Delete collection"
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
