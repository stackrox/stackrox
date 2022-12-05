import React, { ReactElement } from 'react';
import {
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
import { CollectionResponse } from 'services/CollectionsService';
import useCollection from './hooks/useCollection';
import CollectionFormDrawer, { CollectionFormDrawerProps } from './CollectionFormDrawer';
import CollectionLoadError from './CollectionLoadError';

export type CollectionsFormModalProps = {
    hasWriteAccessForCollections: boolean;
    collectionId: string;
    onClose: () => void;
};

function getCollectionTableCells(): ReturnType<
    CollectionFormDrawerProps['getCollectionTableCells']
> {
    return [
        {
            name: 'Name',
            render: ({ id, name }: CollectionResponse) => (
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
    collectionId,
    onClose,
}: CollectionsFormModalProps) {
    const isLargeScreen = useMediaQuery({ query: '(min-width: 992px)' }); // --pf-global--breakpoint--lg
    const {
        isOpen: isDrawerOpen,
        toggleSelect: toggleDrawer,
        closeSelect: closeDrawer,
        openSelect: openDrawer,
    } = useSelectToggle(isLargeScreen);

    const { data, loading, error } = useCollection(collectionId);

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
                // We do not want to present the user with options to change the collection when in this modal
                hasWriteAccessForCollections={false}
                action={{
                    type: 'view',
                    collectionId,
                }}
                collectionData={data}
                isInlineDrawer={isLargeScreen}
                isDrawerOpen={isDrawerOpen}
                toggleDrawer={toggleDrawer}
                // Since the form cannot be submitted, stub this out with an empty promise
                onSubmit={() => Promise.resolve()}
                onCancel={onClose}
                getCollectionTableCells={getCollectionTableCells}
            />
        );
        modalTitle = data.collection.name;
    }

    const modalHeaderButtons =
        error || loading ? (
            ''
        ) : (
            <>
                {hasWriteAccessForCollections && (
                    <Button
                        variant="link"
                        component="a"
                        href={`${collectionsBasePath}/${collectionId}?action=edit`}
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
            <Divider component="div" />
            {content}
        </Modal>
    );
}

export default CollectionsFormModal;
