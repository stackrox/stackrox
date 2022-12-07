import React, { ReactElement, useEffect } from 'react';
import {
    Alert,
    Badge,
    Button,
    EmptyState,
    EmptyStateIcon,
    EmptyStateVariant,
    ExpandableSection,
    ExpandableSectionToggle,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Label,
    TextInput,
    Title,
} from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';
import { TableComposable, TableVariant, Tbody, Tr, Td } from '@patternfly/react-table';
import { useFormik } from 'formik';
import * as yup from 'yup';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { CollectionResponse } from 'services/CollectionsService';
import { getIsValidLabelKey } from 'utils/labels';
import { CollectionPageAction } from './collections.utils';
import RuleSelector from './RuleSelector';
import CollectionAttacher, { CollectionAttacherProps } from './CollectionAttacher';
import {
    Collection,
    ScopedResourceSelector,
    SelectorEntityType,
    selectorEntityTypes,
} from './types';
import { CollectionSaveError } from './errorUtils';

import './CollectionForm.css';

const ruleSectionContentId = 'expandable-rules-section';
const attachmentSectionContentId = 'expandable-attachment-section';

function AttachedCollectionTable({
    collections,
    collectionTableCells,
}: {
    collections: CollectionResponse[];
    collectionTableCells: CollectionAttacherProps['collectionTableCells'];
}) {
    return collections.length > 0 ? (
        <TableComposable aria-label="Attached collections" variant={TableVariant.compact}>
            <Tbody>
                {collections.map((collection) => (
                    <Tr key={collection.name}>
                        {collectionTableCells.map(({ name, render }) => (
                            <Td key={name} dataLabel={name}>
                                {render(collection)}
                            </Td>
                        ))}
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
    onFormChange: (values: Collection) => void;
    onSubmit: (collection: Collection) => Promise<void>;
    onCancel: () => void;
    saveError?: CollectionSaveError | undefined;
    clearSaveError?: () => void;
    /* Table cells to render for each collection in the CollectionAttacher component */
    getCollectionTableCells: (
        collectionErrorId: string | undefined
    ) => CollectionAttacherProps['collectionTableCells'];
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
                          key: yup.string().trim().required().test(getIsValidLabelKey),
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

const validationSchema = yup.object({
    name: yup.string().trim().required(),
    description: yup.string(),
    embeddedCollectionIds: yup.array(yup.string().trim().required()),
    resourceSelector: yup.object().shape({
        Deployment: yupResourceSelectorObject(),
        Namespace: yupResourceSelectorObject(),
        Cluster: yupResourceSelectorObject(),
    }),
});

function getRuleCount(resourceSelector: Collection['resourceSelector']) {
    let count = 0;

    selectorEntityTypes.forEach((entityType) => {
        const selector = resourceSelector[entityType];
        if (selector.type === 'ByName') {
            count += 1;
        } else if (selector.type === 'ByLabel') {
            count += selector.rules.length;
        }
    });

    return count;
}

function CollectionForm({
    hasWriteAccessForCollections,
    action,
    initialData,
    initialEmbeddedCollections,
    saveError,
    clearSaveError = () => {},
    onFormChange,
    onSubmit,
    onCancel,
    getCollectionTableCells,
}: CollectionFormProps) {
    const isReadOnly = action.type === 'view' || !hasWriteAccessForCollections;

    const { isOpen: isRuleSectionOpen, onToggle: ruleSectionOnToggle } = useSelectToggle(true);
    const { isOpen: isAttachmentSectionOpen, onToggle: attachmentSectionOnToggle } =
        useSelectToggle(true);

    const {
        values,
        errors: formikErrors,
        touched,
        handleChange,
        handleBlur,
        setFieldValue,
        submitForm,
        isSubmitting,
    } = useFormik({
        initialValues: initialData,
        onSubmit: (collection, { setSubmitting }) => {
            onSubmit(collection).catch(() => {
                setSubmitting(false);
            });
        },
        validationSchema,
    });

    useEffect(() => {
        if (saveError) {
            return;
        }
        // We need to manually validate the values onChange here since Formik update the values
        // before validation and the declarative state has `isValid === true` and `isValidation == false`
        // at the same time that the actual value object is invalid.
        validationSchema
            .validate(values)
            .then(() => onFormChange(values))
            .catch(() => {
                /* Validation failed, do not propagate change */
            });
    }, [values, saveError, onFormChange]);

    // Synchronize the value of "name" in the form field when the page action changes
    // e.g. from 'view' -> 'clone'
    useEffect(() => {
        const nameValue = {
            create: '',
            view: initialData.name,
            edit: initialData.name,
            clone: `${initialData.name} (COPY)`,
        }[action.type];

        setFieldValue('name', nameValue).catch(() => {
            // Nothing to do on error
        });
    }, [action.type, initialData.name, setFieldValue]);

    const errors = {
        ...formikErrors,
    };

    // We can associate this type of server error to a specific field, so update the formik errors
    if (saveError?.type === 'DuplicateName') {
        errors.name = saveError.message;
    }

    const collectionTableCells = getCollectionTableCells(
        saveError?.type === 'CollectionLoop' ? saveError.loopId : undefined
    );

    const onResourceSelectorChange = (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector
    ) => setFieldValue(`resourceSelector.${entityType}`, scopedResourceSelector);

    const onEmbeddedCollectionsChange = (newCollections: CollectionResponse[]) => {
        if (
            saveError?.type === 'CollectionLoop' &&
            !newCollections.find(({ id }) => id === saveError.loopId)
        ) {
            clearSaveError();
        }
        return setFieldValue(
            'embeddedCollectionIds',
            newCollections.map(({ id }) => id)
        );
    };

    const ruleCount = getRuleCount(values.resourceSelector);

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
                            <FormGroup
                                label="Name"
                                fieldId="name"
                                isRequired={!isReadOnly}
                                helperTextInvalid={errors.name}
                                validated={errors.name && touched.name ? 'error' : 'default'}
                            >
                                <TextInput
                                    id="name"
                                    name="name"
                                    value={values.name}
                                    validated={errors.name && touched.name ? 'error' : 'default'}
                                    onChange={(_, e) => {
                                        if (saveError?.type === 'DuplicateName') {
                                            clearSaveError();
                                        }
                                        handleChange(e);
                                    }}
                                    onBlur={handleBlur}
                                    readOnlyVariant={isReadOnly ? 'plain' : undefined}
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
                                    readOnlyVariant={isReadOnly ? 'plain' : undefined}
                                />
                            </FormGroup>
                        </FlexItem>
                    </Flex>
                </Flex>
                <div className="collection-form-expandable-section">
                    <ExpandableSectionToggle
                        contentId={ruleSectionContentId}
                        isExpanded={isRuleSectionOpen}
                        onToggle={ruleSectionOnToggle}
                    >
                        <Flex
                            alignItems={{ default: 'alignItemsCenter' }}
                            spaceItems={{ default: 'spaceItemsSm' }}
                        >
                            <Title
                                className={isReadOnly ? 'pf-u-mb-0' : 'pf-u-mb-xs'}
                                headingLevel="h2"
                            >
                                Collection rules
                            </Title>
                            <Badge isRead>{ruleCount}</Badge>
                        </Flex>
                        {!isReadOnly && (
                            <p>
                                Select deployments via rules. You can use regular expressions (RE2
                                syntax).
                            </p>
                        )}
                    </ExpandableSectionToggle>

                    <ExpandableSection
                        isDetached
                        contentId={ruleSectionContentId}
                        isExpanded={isRuleSectionOpen}
                    >
                        <Flex
                            className="pf-u-p-md"
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsMd' }}
                        >
                            {saveError?.type === 'EmptyCollection' && (
                                <Alert
                                    title="At least one rule must be configured or one collection must be attached from the section below"
                                    variant="danger"
                                    isInline
                                />
                            )}
                            {saveError?.type === 'InvalidRule' && (
                                <Alert title={saveError.message} variant="danger" isInline>
                                    {saveError.details}
                                </Alert>
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
                    </ExpandableSection>
                </div>

                <div className="collection-form-expandable-section">
                    <ExpandableSectionToggle
                        contentId={attachmentSectionContentId}
                        isExpanded={isAttachmentSectionOpen}
                        onToggle={attachmentSectionOnToggle}
                    >
                        <Flex
                            alignItems={{ default: 'alignItemsCenter' }}
                            spaceItems={{ default: 'spaceItemsSm' }}
                        >
                            <Title className="pf-u-mb-xs" headingLevel="h2">
                                Attached collections
                            </Title>
                            <Badge isRead>{values.embeddedCollectionIds.length}</Badge>
                        </Flex>
                        {!isReadOnly && <p>Extend this collection by attaching other sets.</p>}
                    </ExpandableSectionToggle>

                    <ExpandableSection
                        isDetached
                        contentId={attachmentSectionContentId}
                        isExpanded={isAttachmentSectionOpen}
                    >
                        <Flex
                            className="pf-u-p-md"
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsMd' }}
                        >
                            {saveError?.type === 'EmptyCollection' && (
                                <Alert
                                    title="At least one collection must be attached or one rule must be configured from the section above"
                                    variant="danger"
                                    isInline
                                />
                            )}
                            {saveError?.type === 'CollectionLoop' && (
                                <Alert title={saveError.message} variant="danger" isInline>
                                    {saveError.details}
                                </Alert>
                            )}
                            {isReadOnly ? (
                                <AttachedCollectionTable
                                    collections={initialEmbeddedCollections}
                                    collectionTableCells={collectionTableCells}
                                />
                            ) : (
                                <>
                                    <CollectionAttacher
                                        excludedCollectionId={
                                            action.type === 'edit' ? action.collectionId : null
                                        }
                                        initialEmbeddedCollections={initialEmbeddedCollections}
                                        onSelectionChange={onEmbeddedCollectionsChange}
                                        collectionTableCells={collectionTableCells}
                                    />
                                </>
                            )}
                        </Flex>
                    </ExpandableSection>
                </div>
            </Flex>
            {action.type !== 'view' && (
                <div className="pf-u-background-color-100 pf-u-p-lg pf-u-py-md">
                    <Button
                        className="pf-u-mr-md"
                        onClick={submitForm}
                        isDisabled={isSubmitting}
                        isLoading={isSubmitting}
                    >
                        Save
                    </Button>
                    <Button variant="secondary" isDisabled={isSubmitting} onClick={onCancel}>
                        Cancel
                    </Button>
                </div>
            )}
        </Form>
    );
}

export default CollectionForm;
