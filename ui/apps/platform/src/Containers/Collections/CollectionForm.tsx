import React, { ReactElement, useEffect } from 'react';
import { useHistory } from 'react-router-dom';
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
import { CubesIcon } from '@patternfly/react-icons';
import { TableComposable, TableVariant, Tbody, Tr, Td } from '@patternfly/react-table';
import { Formik, FormikHelpers } from 'formik';
import * as yup from 'yup';

import { collectionsBasePath } from 'routePaths';
import { CollectionResponse } from 'services/CollectionsService';
import { CollectionPageAction } from './collections.utils';
import RuleSelector from './RuleSelector';
import CollectionAttacher from './CollectionAttacher';
import CollectionResults from './CollectionResults';
import { Collection, ScopedResourceSelector, SelectorEntityType } from './types';
import { parseCollection } from './converter';

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
    /* initial, unparsed data used to populate the form */
    collectionData: {
        collection: Omit<CollectionResponse, 'id'>;
        embeddedCollections: CollectionResponse[];
    };
    /* Whether or not to display the collection results in an inline drawer. If false, will
    display collection results in an overlay drawer. */
    useInlineDrawer: boolean;
    isDrawerOpen: boolean;
    toggleDrawer: (isOpen: boolean) => void;
    onSubmit: (collection: Collection) => Promise<void>;
    /* Callback used when clicking on a collection name in the CollectionAttacher section. If
    left undefined, collection names will not be linked. */
    appendTableLinkAction?: (collectionId: string) => void;
    /* content to render before the main form */
    headerContent?: ReactElement;
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
    collectionData,
    useInlineDrawer,
    isDrawerOpen,
    toggleDrawer,
    onSubmit,
    headerContent,
}: CollectionFormProps) {
    const history = useHistory();

    const initialData = parseCollection(collectionData.collection);
    const initialEmbeddedCollections = collectionData.embeddedCollections;

    useEffect(() => {
        toggleDrawer(useInlineDrawer);
    }, [toggleDrawer, useInlineDrawer]);

    const isReadOnly = action.type === 'view' || !hasWriteAccessForCollections;

    function onCancelSave() {
        history.push({ pathname: `${collectionsBasePath}` });
    }

    const onResourceSelectorChange =
        (setFieldValue: FormikHelpers<Collection>['setFieldValue']) =>
        (entityType: SelectorEntityType, scopedResourceSelector: ScopedResourceSelector) =>
            setFieldValue(`resourceSelector.${entityType}`, scopedResourceSelector);

    const onEmbeddedCollectionsChange =
        (setFieldValue: FormikHelpers<Collection>['setFieldValue']) =>
        (newCollections: CollectionResponse[]) =>
            setFieldValue(
                'embeddedCollectionIds',
                newCollections.map(({ id }) => id)
            );

    return (
        <>
            <Drawer isExpanded={isDrawerOpen} isInline={useInlineDrawer}>
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
                                    <DrawerCloseButton onClick={() => toggleDrawer(false)} />
                                </DrawerActions>
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
                            <Formik
                                initialValues={
                                    action.type === 'clone'
                                        ? { ...initialData, name: `${initialData.name} (COPY)` }
                                        : initialData
                                }
                                onSubmit={(collection, { setSubmitting }) => {
                                    onSubmit(collection).catch(() => {
                                        setSubmitting(false);
                                    });
                                }}
                                validationSchema={yup.object({
                                    name: yup.string().trim().required(),
                                    description: yup.string(),
                                    embeddedCollectionIds: yup.array(
                                        yup.string().trim().required()
                                    ),
                                    resourceSelector: yup.object().shape({
                                        Deployment: yupResourceSelectorObject(),
                                        Namespace: yupResourceSelectorObject(),
                                        Cluster: yupResourceSelectorObject(),
                                    }),
                                })}
                            >
                                {({
                                    values,
                                    isValid,
                                    errors,
                                    handleChange,
                                    handleBlur,
                                    setFieldValue,
                                    submitForm,
                                    isSubmitting,
                                }) => (
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
                                                        <FormGroup
                                                            label="Name"
                                                            fieldId="name"
                                                            isRequired
                                                        >
                                                            <TextInput
                                                                id="name"
                                                                name="name"
                                                                value={values.name}
                                                                validated={
                                                                    errors.name
                                                                        ? 'error'
                                                                        : 'default'
                                                                }
                                                                onChange={(_, e) => handleChange(e)}
                                                                onBlur={handleBlur}
                                                                isDisabled={isReadOnly}
                                                            />
                                                        </FormGroup>
                                                    </FlexItem>
                                                    <FlexItem flex={{ default: 'flex_2' }}>
                                                        <FormGroup
                                                            label="Description"
                                                            fieldId="description"
                                                        >
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
                                                    className={
                                                        isReadOnly ? 'pf-u-mb-md' : 'pf-u-mb-xs'
                                                    }
                                                    headingLevel="h2"
                                                >
                                                    Collection rules
                                                </Title>
                                                {!isReadOnly && (
                                                    <>
                                                        <p>
                                                            Select deployments via rules. You can
                                                            use regular expressions (RE2 syntax).
                                                        </p>
                                                    </>
                                                )}
                                                <RuleSelector
                                                    entityType="Deployment"
                                                    scopedResourceSelector={
                                                        values.resourceSelector.Deployment
                                                    }
                                                    handleChange={onResourceSelectorChange(
                                                        setFieldValue
                                                    )}
                                                    validationErrors={
                                                        errors.resourceSelector?.Deployment
                                                    }
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
                                                    scopedResourceSelector={
                                                        values.resourceSelector.Namespace
                                                    }
                                                    handleChange={onResourceSelectorChange(
                                                        setFieldValue
                                                    )}
                                                    validationErrors={
                                                        errors.resourceSelector?.Namespace
                                                    }
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
                                                    scopedResourceSelector={
                                                        values.resourceSelector.Cluster
                                                    }
                                                    handleChange={onResourceSelectorChange(
                                                        setFieldValue
                                                    )}
                                                    validationErrors={
                                                        errors.resourceSelector?.Cluster
                                                    }
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
                                                        <p>
                                                            Extend this collection by attaching
                                                            other sets.
                                                        </p>
                                                        <CollectionAttacher
                                                            initialEmbeddedCollections={
                                                                initialEmbeddedCollections
                                                            }
                                                            onSelectionChange={onEmbeddedCollectionsChange(
                                                                setFieldValue
                                                            )}
                                                        />
                                                    </>
                                                )}
                                            </Flex>
                                        </Flex>
                                        {action.type !== 'view' && (
                                            <div className="pf-u-background-color-100 pf-u-p-lg pf-u-py-md">
                                                <Button
                                                    className="pf-u-mr-md"
                                                    onClick={submitForm}
                                                    isDisabled={isSubmitting || !isValid}
                                                    isLoading={isSubmitting}
                                                >
                                                    Save
                                                </Button>
                                                <Button
                                                    variant="secondary"
                                                    isDisabled={isSubmitting}
                                                    onClick={onCancelSave}
                                                >
                                                    Cancel
                                                </Button>
                                            </div>
                                        )}
                                    </Form>
                                )}
                            </Formik>
                        )}
                    </DrawerContentBody>
                </DrawerContent>
            </Drawer>
        </>
    );
}

export default CollectionForm;
