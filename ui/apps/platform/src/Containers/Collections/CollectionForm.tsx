import React, { useEffect, useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
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
    Dropdown,
    DropdownItem,
    DropdownSeparator,
    DropdownToggle,
    Flex,
    FlexItem,
    Form,
    Label,
    Stack,
    Text,
    Title,
} from '@patternfly/react-core';
import { CaretDownIcon } from '@patternfly/react-icons';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useToasts, { Toast } from 'hooks/patternfly/useToasts';
import { collectionsBasePath } from 'routePaths';
import { deleteCollection } from 'services/CollectionsService';
import { Formik, FormikHelpers, useFormikContext } from 'formik';
import { CollectionPageAction } from './collections.utils';
import RuleSelector from './RuleSelector';
import CollectionAttacher from './CollectionAttacher';
import CollectionResults from './CollectionResults';
import { Collection, ScopedResourceSelector, SelectorEntityType } from './types';

const FormikWatcher = ({ onChange }) => {
    const { values, isValid } = useFormikContext();

    useEffect(() => {
        //  TODO Beware that this fires twice
        onChange(values, isValid);
    }, [onChange, isValid, values]);

    return null;
};

export type CollectionFormProps = {
    hasWriteAccessForCollections: boolean;
    /* The user's workflow action for this collection */
    action: CollectionPageAction;
    /* initial data used to populate the form */
    initialData: Collection;
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
    hasWriteAccessForCollections,
    action,
    initialData,
    useInlineDrawer,
    showBreadcrumbs,
}: CollectionFormProps) {
    const history = useHistory();

    const {
        isOpen: drawerIsOpen,
        toggleSelect: toggleDrawer,
        closeSelect: closeDrawer,
        openSelect: openDrawer,
    } = useSelectToggle(useInlineDrawer);
    const {
        isOpen: menuIsOpen,
        toggleSelect: toggleMenu,
        closeSelect: closeMenu,
    } = useSelectToggle();
    const [deleteId, setDeleteId] = useState<string | null>(null);
    const [isDeleting, setIsDeleting] = useState(false);
    const { toasts, addToast, removeToast } = useToasts();

    useEffect(() => {
        toggleDrawer(useInlineDrawer);
    }, [toggleDrawer, useInlineDrawer]);

    const pageTitle = action.type === 'create' ? 'Create collection' : initialData.name;

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
                    `Could not delete collection ${initialData.name ?? ''}`,
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

    function onRuleSelectorChange(setFieldValue: FormikHelpers<unknown>['setFieldValue']) {
        return (
            entityType: SelectorEntityType,
            scopedResourceSelector: ScopedResourceSelector | null
        ) => setFieldValue(`selectorRules.${entityType}`, scopedResourceSelector);
    }

    return (
        <>
            <Drawer isExpanded={drawerIsOpen} isInline={useInlineDrawer}>
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
                            <DrawerPanelBody className="pf-u-h-100" style={{ overflow: 'auto' }}>
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
                                {action.type === 'view' && hasWriteAccessForCollections && (
                                    <>
                                        <Dropdown
                                            onSelect={closeMenu}
                                            position="right"
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
                                                        onEditCollection(action.collectionId)
                                                    }
                                                >
                                                    Edit collection
                                                </DropdownItem>,
                                                <DropdownItem
                                                    key="Clone collection"
                                                    component="button"
                                                    onClick={() =>
                                                        onCloneCollection(action.collectionId)
                                                    }
                                                >
                                                    Clone collection
                                                </DropdownItem>,
                                                <DropdownSeparator key="Separator" />,
                                                <DropdownItem
                                                    key="Delete collection"
                                                    component="button"
                                                    isDisabled={initialData.inUse}
                                                    onClick={() => setDeleteId(action.collectionId)}
                                                >
                                                    {initialData.inUse
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
                                {drawerIsOpen ? (
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
                        <Formik initialValues={initialData} onSubmit={() => {}}>
                            {({ values, handleChange, setFieldValue }) => (
                                <Form>
                                    <FormikWatcher
                                        onChange={(formValue: Collection) =>
                                            console.log('formik change', formValue.selectorRules)
                                        }
                                    />
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

                                        <div className="pf-u-background-color-100 pf-u-p-lg">
                                            <Flex
                                                direction={{ default: 'column' }}
                                                spaceItems={{ default: 'spaceItemsMd' }}
                                            >
                                                <Title headingLevel="h2">
                                                    Add new collection rules
                                                </Title>
                                                <RuleSelector
                                                    entityType="Deployment"
                                                    scopedResourceSelector={
                                                        values.selectorRules.Deployment
                                                    }
                                                    onOptionChange={onRuleSelectorChange(
                                                        setFieldValue
                                                    )}
                                                />
                                                <Label
                                                    variant="outline"
                                                    isCompact
                                                    className="pf-u-align-self-center"
                                                >
                                                    in
                                                </Label>
                                                <RuleSelector
                                                    entityType="Namespace"
                                                    scopedResourceSelector={
                                                        values.selectorRules.Namespace
                                                    }
                                                    onOptionChange={onRuleSelectorChange(
                                                        setFieldValue
                                                    )}
                                                />
                                                <Label
                                                    variant="outline"
                                                    isCompact
                                                    className="pf-u-align-self-center"
                                                >
                                                    in
                                                </Label>
                                                <RuleSelector
                                                    entityType="Cluster"
                                                    scopedResourceSelector={
                                                        values.selectorRules.Cluster
                                                    }
                                                    onOptionChange={onRuleSelectorChange(
                                                        setFieldValue
                                                    )}
                                                />
                                            </Flex>
                                        </div>
                                        <Card>
                                            <CardBody>
                                                <Title headingLevel="h2">
                                                    Attach existing collections
                                                </Title>
                                                <CollectionAttacher />
                                            </CardBody>
                                        </Card>
                                    </Flex>
                                    {action.type !== 'view' && (
                                        <div className="pf-u-p-lg pf-u-py-md">
                                            <>
                                                <Button className="pf-u-mr-md">Save</Button>
                                                <Button variant="secondary">Cancel</Button>
                                            </>
                                        </div>
                                    )}
                                </Form>
                            )}
                        </Formik>
                    </DrawerContentBody>
                </DrawerContent>
            </Drawer>
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }: Toast) => (
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

export default CollectionForm;
