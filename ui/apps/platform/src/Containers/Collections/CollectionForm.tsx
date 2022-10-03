import React, { useEffect } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Button,
    Card,
    CardBody,
    Divider,
    Drawer,
    DrawerActions,
    DrawerCloseButton,
    DrawerContent,
    DrawerContentBody,
    DrawerHead,
    DrawerPanelBody,
    DrawerPanelContent,
    Flex,
    FlexItem,
    Text,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { collectionsBasePath } from 'routePaths';
import { ResolvedCollectionResponse } from 'services/CollectionsService';
import { CollectionPageAction } from './collections.utils';
import RuleSelector from './RuleSelector';
import CollectionAttacher from './CollectionAttacher';
import CollectionResults from './CollectionResults';

export type CollectionFormProps = {
    /* The user's workflow action for this collection */
    action: CollectionPageAction['type'];
    /* initial data used to populate the form, or `undefined` in the case of a new collection */
    initialData: ResolvedCollectionResponse | undefined;
    /* Whether or not to display the collection results in an inline drawer. If false, will
    display collection results in an overlay drawer. */
    useInlineDrawer: boolean;
    /* Whether or not to show breadcrumb navigation at the top of the form */
    showBreadcrumbs: boolean;
    /* Callback used when clicking on a collection name in the CollectionAttacher section. If
    left undefined, collection names will not be linked. */
    appendTableLinkAction?: (collectionId: string) => void;
};

function CollectionForm({
    action,
    initialData,
    useInlineDrawer,
    showBreadcrumbs,
}: CollectionFormProps) {
    const {
        isOpen,
        closeSelect: closeDrawer,
        openSelect: openDrawer,
        toggleSelect: toggleDrawer,
    } = useSelectToggle(useInlineDrawer);

    useEffect(() => {
        toggleDrawer(useInlineDrawer);
    }, [toggleDrawer, useInlineDrawer]);

    const pageTitle = initialData ? initialData.collection.name : 'Create collection';

    return (
        <>
            <Drawer isExpanded={isOpen} isInline={useInlineDrawer}>
                <DrawerContent
                    panelContent={
                        <DrawerPanelContent
                            style={{
                                borderLeft: 'var(--pf-global--BorderColor--100) 1px solid',
                            }}
                        >
                            <DrawerHead>
                                <Title headingLevel="h2">Collection results</Title>
                                <Text>See a live preview of current matches.</Text>
                                <DrawerActions>
                                    <DrawerCloseButton onClick={closeDrawer} />
                                </DrawerActions>
                            </DrawerHead>
                            <DrawerPanelBody style={{ overflow: 'auto', height: '100%' }}>
                                <CollectionResults />
                            </DrawerPanelBody>
                        </DrawerPanelContent>
                    }
                >
                    <DrawerContentBody className="pf-u-background-color-100 pf-u-display-flex pf-u-flex-direction-column">
                        {showBreadcrumbs && (
                            <>
                                <Breadcrumb className="pf-u-my-xs pf-u-px-lg pf-u-py-md">
                                    <BreadcrumbItemLink to={collectionsBasePath}>
                                        Collections
                                    </BreadcrumbItemLink>
                                    <BreadcrumbItem>{pageTitle}</BreadcrumbItem>
                                </Breadcrumb>
                                <Divider component="div" />
                            </>
                        )}
                        <Flex className="pf-u-p-lg" alignItems={{ default: 'alignItemsCenter' }}>
                            <FlexItem flex={{ default: 'flex_1' }}>
                                <Title headingLevel="h1">{pageTitle}</Title>
                            </FlexItem>
                            <FlexItem align={{ default: 'alignRight' }}>
                                {isOpen ? (
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
                        <Flex
                            className="pf-u-background-color-200 pf-u-p-lg"
                            spaceItems={{ default: 'spaceItemsMd' }}
                            direction={{ default: 'column' }}
                        >
                            <Card>
                                <CardBody>
                                    <Title headingLevel="h2">Collection details</Title>
                                </CardBody>
                            </Card>
                            <Card>
                                <CardBody>
                                    <Title headingLevel="h2">Add new collection rules</Title>
                                    <RuleSelector />
                                    <RuleSelector />
                                    <RuleSelector />
                                </CardBody>
                            </Card>
                            <Card>
                                <CardBody>
                                    <Title headingLevel="h2">Attach existing collections</Title>
                                    <CollectionAttacher />
                                </CardBody>
                            </Card>
                        </Flex>

                        <div className="pf-u-p-lg pf-u-py-md">
                            <Button className="pf-u-mr-md">
                                {action === 'view' ? 'Edit' : 'Save'}
                            </Button>
                            <Button variant="secondary">Cancel</Button>
                        </div>
                    </DrawerContentBody>
                </DrawerContent>
            </Drawer>
        </>
    );
}

export default CollectionForm;
