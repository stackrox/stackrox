import React, { ReactElement, useEffect } from 'react';
import {
    Drawer,
    DrawerActions,
    DrawerCloseButton,
    DrawerContent,
    DrawerContentBody,
    DrawerHead,
    DrawerPanelBody,
    DrawerPanelContent,
    Text,
    Title,
} from '@patternfly/react-core';

import { CollectionResponse } from 'services/CollectionsService';
import { CollectionPageAction } from './collections.utils';
import CollectionResults from './CollectionResults';
import { Collection } from './types';
import { parseCollection } from './converter';
import CollectionForm from './CollectionForm';

export type CollectionFormDrawerProps = {
    hasWriteAccessForCollections: boolean;
    /* The user's workflow action for this collection */
    action: CollectionPageAction;
    collectionData: {
        collection: Omit<CollectionResponse, 'id'>;
        embeddedCollections: CollectionResponse[];
    };
    /* Whether or not to display the collection results in an inline drawer. If false, will
    display collection results in an overlay drawer. */
    isInlineDrawer: boolean;
    isDrawerOpen: boolean;
    toggleDrawer: (isOpen: boolean) => void;
    headerContent?: ReactElement;
    onSubmit: (collection: Collection) => Promise<void>;
    /* Callback used when clicking on a collection name in the CollectionAttacher section. */
    appendTableLinkAction: (collectionId: string) => void;
};

function CollectionFormDrawer({
    hasWriteAccessForCollections,
    action,
    collectionData,
    headerContent,
    isInlineDrawer,
    isDrawerOpen,
    toggleDrawer,
    onSubmit,
    appendTableLinkAction,
}: CollectionFormDrawerProps) {
    const initialData = parseCollection(collectionData.collection);
    const initialEmbeddedCollections = collectionData.embeddedCollections;

    useEffect(() => {
        toggleDrawer(isInlineDrawer);
    }, [toggleDrawer, isInlineDrawer]);

    return (
        <>
            <Drawer isExpanded={isDrawerOpen} isInline={isInlineDrawer}>
                <DrawerContent
                    panelContent={
                        <DrawerPanelContent
                            style={{
                                borderLeft: 'var(--pf-global--BorderColor--100) 1px solid',
                            }}
                        >
                            <DrawerHead>
                                <Title headingLevel="h2">Collection results</Title>
                                <Text>See a preview of current matches.</Text>
                                {!isInlineDrawer && (
                                    <DrawerActions>
                                        <DrawerCloseButton onClick={() => toggleDrawer(false)} />
                                    </DrawerActions>
                                )}
                            </DrawerHead>
                            <DrawerPanelBody className="pf-u-h-100" style={{ overflow: 'auto' }}>
                                <CollectionResults />
                            </DrawerPanelBody>
                        </DrawerPanelContent>
                    }
                >
                    <DrawerContentBody className="pf-u-background-color-100 pf-u-display-flex pf-u-flex-direction-column">
                        {headerContent}
                        {initialData instanceof AggregateError ? (
                            <>
                                {initialData.errors}
                                {/* TODO - Handle inline UI for unsupported rule errors */}
                            </>
                        ) : (
                            <CollectionForm
                                hasWriteAccessForCollections={hasWriteAccessForCollections}
                                action={action}
                                initialData={initialData}
                                initialEmbeddedCollections={initialEmbeddedCollections}
                                onSubmit={onSubmit}
                                appendTableLinkAction={appendTableLinkAction}
                            />
                        )}
                    </DrawerContentBody>
                </DrawerContent>
            </Drawer>
        </>
    );
}

export default CollectionFormDrawer;
