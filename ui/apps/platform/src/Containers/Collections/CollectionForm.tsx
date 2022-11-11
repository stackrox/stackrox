import React, { ReactElement } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Button,
    EmptyState,
    EmptyStateIcon,
    EmptyStateVariant,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Label,
    TextInput,
    Title,
    Truncate,
} from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';
import { TableComposable, TableVariant, Tbody, Tr, Td } from '@patternfly/react-table';
import { useFormik } from 'formik';
import * as yup from 'yup';

import { collectionsBasePath } from 'routePaths';
import { CollectionResponse } from 'services/CollectionsService';
import { CollectionPageAction } from './collections.utils';
import RuleSelector from './RuleSelector';
import CollectionAttacher from './CollectionAttacher';
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
    /* parsed collection data used to populate the form */
    initialData: Collection;
    /* collection responses for the embedded collections of `initialData` */
    initialEmbeddedCollections: CollectionResponse[];
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
    initialData,
    initialEmbeddedCollections,
    onSubmit,
}: CollectionFormProps) {
    const history = useHistory();

    const isReadOnly = action.type === 'view' || !hasWriteAccessForCollections;

    const {
        values,
        isValid,
        errors,
        handleChange,
        handleBlur,
        setFieldValue,
        submitForm,
        isSubmitting,
    } = useFormik({
        initialValues:
            action.type === 'clone'
                ? { ...initialData, name: `${initialData.name} (COPY)` }
                : initialData,
        onSubmit: (collection, { setSubmitting }) => {
            onSubmit(collection).catch(() => {
                setSubmitting(false);
            });
        },
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

    function onCancelSave() {
        history.push({ pathname: `${collectionsBasePath}` });
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
                    <Title className={isReadOnly ? 'pf-u-mb-md' : 'pf-u-mb-xs'} headingLevel="h2">
                        Collection rules
                    </Title>
                    {!isReadOnly && (
                        <>
                            <p>
                                Select deployments via rules. You can use regular expressions (RE2
                                syntax).
                            </p>
                        </>
                    )}
                    <RuleSelector
                        collection={values}
                        entityType="Deployment"
                        scopedResourceSelector={values.resourceSelector.Deployment}
                        handleChange={onResourceSelectorChange}
                        validationErrors={errors.resourceSelector?.Deployment}
                        isDisabled={isReadOnly}
                    />
                    <Label variant="outline" isCompact className="pf-u-align-self-center">
                        in
                    </Label>
                    <RuleSelector
                        collection={values}
                        entityType="Namespace"
                        scopedResourceSelector={values.resourceSelector.Namespace}
                        handleChange={onResourceSelectorChange}
                        validationErrors={errors.resourceSelector?.Namespace}
                        isDisabled={isReadOnly}
                    />
                    <Label variant="outline" isCompact className="pf-u-align-self-center">
                        in
                    </Label>
                    <RuleSelector
                        collection={values}
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
                        <AttachedCollectionTable collections={initialEmbeddedCollections} />
                    ) : (
                        <>
                            <p>Extend this collection by attaching other sets.</p>
                            <CollectionAttacher
                                initialEmbeddedCollections={initialEmbeddedCollections}
                                onSelectionChange={onEmbeddedCollectionsChange}
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
                    <Button variant="secondary" isDisabled={isSubmitting} onClick={onCancelSave}>
                        Cancel
                    </Button>
                </div>
            )}
        </Form>
    );
}

export default CollectionForm;
