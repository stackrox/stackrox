import React, { ReactElement } from 'react';
import { Button, Divider, Flex, FlexItem, Modal, Title } from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';
import { useMediaQuery } from 'react-responsive';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { collectionsBasePath } from 'routePaths';
import useCollection from './hooks/useCollection';
import CollectionFormDrawer from './CollectionFormDrawer';

export type CollectionsFormModalProps = {
    hasWriteAccessForCollections: boolean;
    collectionId: string;
    onClose: () => void;
};

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
        content = (
            <Modal
                isOpen
                onClose={onClose}
                aria-label={`View ${data.collection.name}`}
                width="90vw"
                hasNoBodyWrapper
                header={
                    <Flex className="pf-u-pb-md" alignItems={{ default: 'alignItemsCenter' }}>
                        <FlexItem grow={{ default: 'grow' }}>
                            <Title headingLevel="h2">{data.collection.name}</Title>
                        </FlexItem>
                        {hasWriteAccessForCollections && (
                            <Button
                                variant="link"
                                component="a"
                                href={`${collectionsBasePath}/${collectionId}?action=edit`}
                                target="_blank"
                                rel="noopener noreferrer"
                                icon={<ExternalLinkAltIcon />}
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
                    </Flex>
                }
            >
                <Divider component="div" />
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
                    appendTableLinkAction={(id) => {
                        const url = `${window.location.origin}${collectionsBasePath}/${id}`;
                        window.open(url, '_blank', 'noopener noreferrer');
                    }}
                />
            </Modal>
        );
    }

    return content;
}

export default CollectionsFormModal;
