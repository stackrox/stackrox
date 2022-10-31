import React, { useEffect, useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    Breadcrumb,
    BreadcrumbItem,
    Button,
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
    EmptyState,
    EmptyStateIcon,
    EmptyStateVariant,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Label,
    Text,
    TextInput,
    Title,
    Truncate,
} from '@patternfly/react-core';
import { CaretDownIcon, CubesIcon } from '@patternfly/react-icons';
import { TableComposable, TableVariant, Tbody, Tr, Td } from '@patternfly/react-table';
import { useFormik } from 'formik';
import * as yup from 'yup';
import isEmpty from 'lodash/isEmpty';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useToasts from 'hooks/patternfly/useToasts';
import { collectionsBasePath } from 'routePaths';
import { CollectionResponse, deleteCollection } from 'services/CollectionsService';
import { CollectionPageAction } from './collections.utils';
import RuleSelector from './RuleSelector';
import CollectionAttacher from './CollectionAttacher';
import CollectionResults from './CollectionResults';
import { Collection, ScopedResourceSelector, SelectorEntityType } from './types';

function AttachedCollectionTable({ collections }: { collections: CollectionResponse[] }) {
    return collections.length > 0 ? (
        <TableComposable aria-label="Attached collections" variant={TableVariant.compact}>
            <Tbody>
                {collections.map(({ name, description }) => (
                    <Tr key={name}>
                        <Td dataLabel="Name">
                            <Button variant="link" className="pf-u-pl-0" isInline>
                                {name}
                            </Button>
                        </Td>
                        <Td dataLabel="Description">
                            <Truncate content={description} />
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    ) : (
        <EmptyState variant={EmptyStateVariant.xs}>
            <EmptyStateIcon icon={CubesIcon} />
            <p>There are no other collections attached to this collection</p>
        </EmptyState>
    );
}

export type CollectionFormProps = {
    hasWriteAccessForCollections: boolean;
    /* The user's workflow action for this collection */
    action: CollectionPageAction;
    /* initial data used to populate the form */
    initialData: Collection;
    /* Collection object references for the list of ids in `initialData` */
    initialEmbeddedCollections: CollectionResponse[];
    /* Whether or not to display the collection results in an inline drawer. If false, will
    display collection results in an overlay drawer. */
    useInlineDrawer: boolean;
    /* Whether or not to show breadcrumb navigation at the top of the form */
    showBreadcrumbs: boolean;
    /* Callback used when clicking on a collection name in the CollectionAttacher section. If
    left undefined, collection names will not be linked. */
    appendTableLinkAction?: (collectionId: string) => void;
};

function yupResourceSelectorObject() {
    return yup.lazy((ruleObject) => {
        if (ruleObject.type === 'All') {
            return yup.object().shape({});
        }

        const { field } = ruleObject;
        return typeof field === 'string' && field.endsWith('Label')
            ? yup.object().shape({
                  field: yup.string().required().matches(new RegExp(field)),
                  rules: yup.array().of(
                      yup.object().shape({
                          operator: yup.string().required().matches(/OR/),
                          key: yup.string().trim().required(),
                          values: yup.array().of(yup.string().trim().required()).required(),
                      })
                  ),
              })
            : yup.object().shape({
                  field: yup.string().required().matches(new RegExp(field)),
                  rule: yup.object().shape({
                      operator: yup.string().required().matches(/OR/),
                      values: yup.array().of(yup.string().trim().required()).required(),
                  }),
              });
    });
}

function CollectionForm({
    hasWriteAccessForCollections,
    action,
    initialData,
    initialEmbeddedCollections,
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

    const { values, errors, handleChange, handleBlur, setFieldValue } = useFormik({
        initialValues: initialData,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup.string().trim().required(),
            description: yup.string(),
            embeddedCollectionIds: yup.array(yup.string().trim().required()),
            resourceSelector: yup.object().shape({
                Deployment: yupResourceSelectorObject(),
                Namespace: yupResourceSelectorObject(),
                Cluster: yupResourceSelectorObject(),
            }),
        }),
    });

    useEffect(() => {
        toggleDrawer(useInlineDrawer);
    }, [toggleDrawer, useInlineDrawer]);

    const pageTitle = action.type === 'create' ? 'Create collection' : values.name;
    const isReadOnly = action.type === 'view' || !hasWriteAccessForCollections;

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

    const onResourceSelectorChange = (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector
    ) => setFieldValue(`resourceSelector.${entityType}`, scopedResourceSelector);

    const onEmbeddedCollectionsChange = (newCollections: CollectionResponse[]) =>
        setFieldValue(
            'embeddedCollectionIds',
            newCollections.map(({ id }) => id)
        );

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
                        <Flex
                            className="pf-u-p-lg"
                            direction={{ default: 'column', md: 'row' }}
                            alignItems={{ default: 'alignItemsFlexStart', md: 'alignItemsCenter' }}
                        >
                            <Title className="pf-u-flex-grow-1" headingLevel="h1">
                                {pageTitle}
                            </Title>
                            <FlexItem align={{ default: 'alignLeft', md: 'alignRight' }}>
                                {action.type === 'view' && hasWriteAccessForCollections && (
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
                        <Form className="pf-u-background-color-200">
                            <Flex
                                className="pf-u-p-lg"
                                spaceItems={{ default: 'spaceItemsMd' }}
                                direction={{ default: 'column' }}
                            >
                                <Flex
                                    className="pf-u-background-color-100 pf-u-p-lg"
                                    direction={{ default: 'column' }}
                                    spaceItems={{ default: 'spaceItemsMd' }}
                                >
                                    <Title headingLevel="h2">Collection details</Title>
                                    <Flex direction={{ default: 'column', lg: 'row' }}>
                                        <FlexItem flex={{ default: 'flex_1' }}>
                                            <FormGroup label="Name" fieldId="name" isRequired>
                                                <TextInput
                                                    id="name"
                                                    name="name"
                                                    value={values.name}
                                                    validated={errors.name ? 'error' : 'default'}
                                                    onChange={(_, e) => handleChange(e)}
                                                    onBlur={handleBlur}
                                                    isDisabled={isReadOnly}
                                                />
                                            </FormGroup>
                                        </FlexItem>
                                        <FlexItem flex={{ default: 'flex_2' }}>
                                            <FormGroup label="Description" fieldId="description">
                                                <TextInput
                                                    id="description"
                                                    name="description"
                                                    value={values.description}
                                                    onChange={(_, e) => handleChange(e)}
                                                    onBlur={handleBlur}
                                                    isDisabled={isReadOnly}
                                                />
                                            </FormGroup>
                                        </FlexItem>
                                    </Flex>
                                </Flex>

                                <Flex
                                    className="pf-u-background-color-100 pf-u-p-lg"
                                    direction={{ default: 'column' }}
                                    spaceItems={{ default: 'spaceItemsMd' }}
                                >
                                    <Title
                                        className={isReadOnly ? 'pf-u-mb-md' : 'pf-u-mb-xs'}
                                        headingLevel="h2"
                                    >
                                        Collection rules
                                    </Title>
                                    {!isReadOnly && (
                                        <>
                                            <p>
                                                Select deployments via rules. You can use regular
                                                expressions (RE2 syntax).
                                            </p>
                                        </>
                                    )}
                                    <RuleSelector
                                        entityType="Deployment"
                                        scopedResourceSelector={values.resourceSelector.Deployment}
                                        handleChange={onResourceSelectorChange}
                                        validationErrors={errors.resourceSelector?.Deployment}
                                        isDisabled={isReadOnly}
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
                                        scopedResourceSelector={values.resourceSelector.Namespace}
                                        handleChange={onResourceSelectorChange}
                                        validationErrors={errors.resourceSelector?.Namespace}
                                        isDisabled={isReadOnly}
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
                                        scopedResourceSelector={values.resourceSelector.Cluster}
                                        handleChange={onResourceSelectorChange}
                                        validationErrors={errors.resourceSelector?.Cluster}
                                        isDisabled={isReadOnly}
                                    />
                                </Flex>

                                <Flex
                                    className="pf-u-background-color-100 pf-u-p-lg"
                                    direction={{ default: 'column' }}
                                    spaceItems={{ default: 'spaceItemsMd' }}
                                >
                                    <Title className="pf-u-mb-xs" headingLevel="h2">
                                        Attached collections
                                    </Title>
                                    {isReadOnly ? (
                                        <AttachedCollectionTable
                                            collections={initialEmbeddedCollections}
                                        />
                                    ) : (
                                        <>
                                            <p>Extend this collection by attaching other sets.</p>
                                            <CollectionAttacher
                                                initialEmbeddedCollections={
                                                    initialEmbeddedCollections
                                                }
                                                onSelectionChange={onEmbeddedCollectionsChange}
                                            />
                                        </>
                                    )}
                                </Flex>
                            </Flex>
                            {action.type !== 'view' && (
                                <div className="pf-u-background-color-100 pf-u-p-lg pf-u-py-md">
                                    <Button className="pf-u-mr-md">Save</Button>
                                    <Button variant="secondary">Cancel</Button>
                                </div>
                            )}
                        </Form>
                    </DrawerContentBody>
                </DrawerContent>
            </Drawer>
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

export default CollectionForm;
