import React, { ReactElement, useEffect } from 'react';
import {
    Button,
    Drawer,
    DrawerActions,
    DrawerCloseButton,
    DrawerContent,
    DrawerContentBody,
    DrawerHead,
    DrawerPanelBody,
    DrawerPanelContent,
    Flex,
    Text,
    Title,
} from '@patternfly/react-core';

import { Collection } from 'services/CollectionsService';
import { SyncAltIcon } from '@patternfly/react-icons';
import { CollectionPageAction } from './collections.utils';
import CollectionResults from './CollectionResults';
import { isCollectionParseError, parseCollection } from './converter';
import CollectionForm, { CollectionFormProps } from './CollectionForm';
import UnsupportedCollectionState from './UnsupportedCollectionState';
import useDryRunConfiguration from './hooks/useDryRunConfiguration';

export type CollectionFormDrawerProps = {
    hasWriteAccessForCollections: boolean;
    /* The user's workflow action for this collection */
    action: CollectionPageAction;
    collectionData: {
        collection: Omit<Collection, 'id'>;
        embeddedCollections: Collection[];
    };
    /* Whether or not to display the collection results in an inline drawer. If false, will
    display collection results in an overlay drawer. */
    isInlineDrawer: boolean;
    isDrawerOpen: boolean;
    toggleDrawer: (isOpen: boolean) => void;
    headerContent?: ReactElement;
    onSubmit: CollectionFormProps['onSubmit'];
    onCancel: CollectionFormProps['onCancel'];
    configError?: CollectionFormProps['configError'];
    setConfigError?: CollectionFormProps['setConfigError'];
    getCollectionTableCells: CollectionFormProps['getCollectionTableCells'];
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
    onCancel,
    configError,
    setConfigError,
    getCollectionTableCells,
}: CollectionFormDrawerProps) {
    const initialData = parseCollection(collectionData.collection);
    const initialEmbeddedCollections = collectionData.embeddedCollections;
    const collectionId = action.type !== 'create' ? action.collectionId : undefined;
    const { dryRunConfig, updateDryRunConfig, refreshConfig } = useDryRunConfiguration(
        collectionId,
        collectionData.collection
    );

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
                                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                    <Flex
                                        grow={{ default: 'grow' }}
                                        direction={{ default: 'column' }}
                                        spaceItems={{ default: 'spaceItemsNone' }}
                                    >
                                        <Title headingLevel="h2">Collection results</Title>
                                        <Text>See a preview of current matches.</Text>
                                    </Flex>
                                    <Button
                                        className="pf-u-mr-md"
                                        variant="plain"
                                        onClick={refreshConfig}
                                        title="Refresh results"
                                    >
                                        <SyncAltIcon />
                                    </Button>
                                </Flex>
                                {!isInlineDrawer && (
                                    <DrawerActions>
                                        <DrawerCloseButton onClick={() => toggleDrawer(false)} />
                                    </DrawerActions>
                                )}
                            </DrawerHead>
                            <DrawerPanelBody className="pf-u-h-100" style={{ overflow: 'auto' }}>
                                <CollectionResults
                                    dryRunConfig={dryRunConfig}
                                    configError={configError}
                                    setConfigError={setConfigError}
                                />
                            </DrawerPanelBody>
                        </DrawerPanelContent>
                    }
                >
                    <DrawerContentBody className="pf-u-background-color-100 pf-u-display-flex pf-u-flex-direction-column">
                        {headerContent}
                        {isCollectionParseError(initialData) ? (
                            <UnsupportedCollectionState
                                className="pf-u-pt-xl"
                                errors={initialData.errors}
                            />
                        ) : (
                            <CollectionForm
                                hasWriteAccessForCollections={hasWriteAccessForCollections}
                                action={action}
                                initialData={initialData}
                                initialEmbeddedCollections={initialEmbeddedCollections}
                                onFormChange={updateDryRunConfig}
                                onSubmit={onSubmit}
                                onCancel={onCancel}
                                configError={configError}
                                setConfigError={setConfigError}
                                getCollectionTableCells={getCollectionTableCells}
                            />
                        )}
                    </DrawerContentBody>
                </DrawerContent>
            </Drawer>
        </>
    );
}

export default CollectionFormDrawer;
