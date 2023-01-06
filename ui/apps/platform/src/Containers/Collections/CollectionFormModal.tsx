import React, { ReactElement } from 'react';
import {
    Alert,
    Bullseye,
    Button,
    Divider,
    Flex,
    FlexItem,
    Modal,
    Spinner,
    Title,
    Truncate,
} from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';
import { useMediaQuery } from 'react-responsive';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { collectionsBasePath } from 'routePaths';
import { Collection } from 'services/CollectionsService';
import useCollection from './hooks/useCollection';
import CollectionFormDrawer, { CollectionFormDrawerProps } from './CollectionFormDrawer';
import CollectionLoadError from './CollectionLoadError';
import { CollectionPageAction } from './collections.utils';

export type CollectionsFormModalProps = {
    hasWriteAccessForCollections: boolean;
    modalAction: Extract<
        { type: 'create' } | { type: 'view'; collectionId: string },
        CollectionPageAction
    >;
    onClose: () => void;
    onSubmit?: CollectionFormDrawerProps['onSubmit'];
    configError?: CollectionFormDrawerProps['configError'];
    setConfigError?: CollectionFormDrawerProps['setConfigError'];
};

function getCollectionTableCells(): ReturnType<
    CollectionFormDrawerProps['getCollectionTableCells']
> {
    return [
        {
            name: 'Name',
            render: ({ id, name }: Collection) => (
                <Button
                    variant="link"
                    component="a"
                    isInline
                    href={`${collectionsBasePath}/${id}?action=edit`}
                    target="_blank"
                    rel="noopener noreferrer"
                    icon={<ExternalLinkAltIcon />}
                >
                    {name}
                </Button>
            ),
            width: 25,
        },
        {
            name: 'Description',
            render: ({ description }) => <Truncate content={description} />,
        },
    ];
}

function CollectionsFormModal({
    hasWriteAccessForCollections,
    modalAction,
    onClose,
    onSubmit = () => Promise.resolve(),
    configError,
    setConfigError,
}: CollectionsFormModalProps) {
    const isLargeScreen = useMediaQuery({ query: '(min-width: 992px)' }); // --pf-global--breakpoint--lg
    const {
        isOpen: isDrawerOpen,
        toggleSelect: toggleDrawer,
        closeSelect: closeDrawer,
        openSelect: openDrawer,
    } = useSelectToggle(isLargeScreen);

    const { data, loading, error } = useCollection(
        modalAction.type === 'view' ? modalAction.collectionId : undefined
    );

    let content: ReactElement | null = null;

    let modalTitle = '';

    if (error) {
        content = (
            <Bullseye className="pf-u-p-2xl">
                <CollectionLoadError error={error} />
            </Bullseye>
        );
        modalTitle = 'Collection error';
    } else if (loading) {
        content = (
            <Bullseye className="pf-u-p-2xl">
                <Spinner isSVG />
            </Bullseye>
        );
        modalTitle = 'Loading collection';
    } else if (data) {
        content = (
            <CollectionFormDrawer
                hasWriteAccessForCollections={hasWriteAccessForCollections}
                action={modalAction}
                collectionData={data}
                isInlineDrawer={isLargeScreen}
                isDrawerOpen={isDrawerOpen}
                toggleDrawer={toggleDrawer}
                onSubmit={onSubmit}
                onCancel={onClose}
                configError={configError}
                setConfigError={setConfigError}
                getCollectionTableCells={getCollectionTableCells}
            />
        );
        modalTitle = data.collection.name || 'Create collection';
    }

    const modalHeaderButtons =
        error || loading ? (
            ''
        ) : (
            <>
                {hasWriteAccessForCollections && modalAction.type === 'view' && (
                    <Button
                        variant="link"
                        component="a"
                        href={`${collectionsBasePath}/${modalAction.collectionId}?action=edit`}
                        target="_blank"
                        rel="noopener noreferrer"
                        icon={<ExternalLinkAltIcon />}
                        iconPosition="right"
                    >
                        Edit collection
                    </Button>
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
                <Divider orientation={{ default: 'vertical' }} component="div" />
            </>
        );

    return (
        <Modal
            isOpen
            onClose={onClose}
            aria-label={modalTitle}
            width="90vw"
            hasNoBodyWrapper
            header={
                <Flex className="pf-u-pb-md" alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem grow={{ default: 'grow' }}>
                        <Title headingLevel="h2">{modalTitle}</Title>
                    </FlexItem>
                    {modalHeaderButtons}
                </Flex>
            }
        >
            {configError && (
                <Alert
                    className="pf-u-mx-lg pf-u-mb-md"
                    title={configError.message}
                    variant="danger"
                    isInline
                >
                    {configError.type === 'UnknownError' ? configError.details : ''}
                </Alert>
            )}
            <Divider component="div" />
            {content}
        </Modal>
    );
}

export default CollectionsFormModal;
