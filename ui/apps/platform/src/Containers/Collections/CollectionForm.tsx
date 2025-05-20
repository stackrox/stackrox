import React, { CSSProperties, ReactElement, useEffect } from 'react';
import {
    Alert,
    Badge,
    Button,
    EmptyState,
    EmptyStateIcon,
    ExpandableSection,
    ExpandableSectionToggle,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Label,
    TextInput,
    Title,
    EmptyStateHeader,
    EmptyStateFooter,
    FormHelperText,
    HelperText,
    HelperTextItem,
} from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';
import { Table, Tbody, Tr, Td } from '@patternfly/react-table';
import { useFormik } from 'formik';
import * as yup from 'yup';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { Collection } from 'services/CollectionsService';
import { getIsValidLabelKey, getIsValidLabelValue } from 'utils/labels';
import { ensureExhaustive } from 'utils/type.utils';
import { CollectionPageAction } from './collections.utils';
import RuleSelector from './RuleSelector';
import CollectionAttacher, { CollectionAttacherProps } from './CollectionAttacher';
import {
    byLabelMatchTypes,
    ByLabelResourceSelector,
    byNameMatchType,
    ByNameResourceSelector,
    ClientCollection,
    ScopedResourceSelector,
    SelectorEntityType,
    selectorEntityTypes,
} from './types';
import { CollectionConfigError } from './errorUtils';

import './CollectionForm.css';

const ruleSectionContentId = 'expandable-rules-section-contentId';
const attachmentSectionContentId = 'expandable-attachment-section-contentId';

const ruleSectionToggleId = 'expandable-rules-section-toggleId';
const attachmentSectionToggleId = 'expandable-attachment-section-toggleId';

function AttachedCollectionTable({
    collections,
    collectionTableCells,
}: {
    collections: Collection[];
    collectionTableCells: CollectionAttacherProps['collectionTableCells'];
}): ReactElement {
    return collections.length > 0 ? (
        <Table aria-label="Attached collections">
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
        </Table>
    ) : (
        <EmptyState>
            <EmptyStateHeader icon={<EmptyStateIcon icon={CubesIcon} />} />
            <EmptyStateFooter>
                <p>There are no other collections attached to this collection</p>
            </EmptyStateFooter>
        </EmptyState>
    );
}

export type CollectionFormProps = {
    hasWriteAccessForCollections: boolean;
    /* The user's workflow action for this collection */
    action: CollectionPageAction;
    /* parsed collection data used to populate the form */
    initialData: ClientCollection;
    /* collection responses for the embedded collections of `initialData` */
    initialEmbeddedCollections: Collection[];
    onFormChange: (values: ClientCollection) => void;
    onSubmit: (collection: ClientCollection) => Promise<void>;
    onCancel: () => void;
    configError?: CollectionConfigError | undefined;
    setConfigError?: (configError: CollectionConfigError | undefined) => void;
    /* Table cells to render for each collection in the CollectionAttacher component */
    getCollectionTableCells: (
        collectionErrorId: string | undefined
    ) => CollectionAttacherProps['collectionTableCells'];
};

function yupLabelRuleObject({ field }: ByLabelResourceSelector) {
    return yup.object().shape({
        field: yup.string().required().matches(new RegExp(field)),
        rules: yup.array().of(
            yup.object().shape({
                operator: yup.string().required().matches(/OR/),
                values: yup
                    .array()
                    .of(
                        yup.object().shape({
                            value: yup
                                .string()
                                .required('This field cannot be empty')
                                .test(
                                    'label-value-k8s-format',
                                    'Labels must be valid k8s labels in the form: key=value',
                                    (val) => {
                                        const parts = val.split('=');
                                        if (parts.length !== 2) {
                                            return false;
                                        }
                                        const validKey = getIsValidLabelKey(parts[0]);
                                        const validLabel = getIsValidLabelValue(parts[1]);
                                        return validKey && validLabel;
                                    }
                                ),
                            matchType: yup
                                .string()
                                .required()
                                .matches(new RegExp(byLabelMatchTypes.join('|'))),
                        })
                    )
                    .required(),
            })
        ),
    });
}

function yupNameRuleObject({ field }: ByNameResourceSelector) {
    return yup.object().shape({
        field: yup.string().required().matches(new RegExp(field)),
        rule: yup.object().shape({
            operator: yup.string().required().matches(/OR/),
            values: yup
                .array()
                .of(
                    yup.object().shape({
                        // TODO Add validation for k8s cluster, namespace, and deployment name characters
                        value: yup.string().trim().required('This field cannot be empty'),
                        matchType: yup
                            .string()
                            .required()
                            .matches(new RegExp(byNameMatchType.join('|'))),
                    })
                )
                .required(),
        }),
    });
}

function yupResourceSelectorObject() {
    return yup.lazy((ruleObject: ScopedResourceSelector) => {
        switch (ruleObject.type) {
            case 'NoneSpecified':
                return yup.object().shape({});
            case 'ByName':
                return yupNameRuleObject(ruleObject);
            case 'ByLabel':
                return yupLabelRuleObject(ruleObject);
            default:
                return ensureExhaustive(ruleObject);
        }
    });
}

const validationSchema = yup.object({
    name: yup
        .string()
        .test(
            'name-is-trimmed',
            'Leading and trailing spaces are not allowed in collection names',
            (name) => name?.trim() === name
        )
        .matches(
            /^[a-zA-Z0-9 <>.-]*$/,
            'Only the following characters are allowed in collection names: a-z A-Z 0-9 < . - >'
        )
        .required(),
    description: yup.string(),
    embeddedCollectionIds: yup.array(yup.string().trim().required()),
    resourceSelector: yup.object().shape({
        Deployment: yupResourceSelectorObject(),
        Namespace: yupResourceSelectorObject(),
        Cluster: yupResourceSelectorObject(),
    }),
});

function getRuleCount(resourceSelector: ClientCollection['resourceSelector']) {
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
    configError,
    setConfigError = () => {},
    onFormChange,
    onSubmit,
    onCancel,
    getCollectionTableCells,
}: CollectionFormProps): ReactElement {
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
        isValid,
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
        onFormChange(values);
    }, [onFormChange, values]);

    // Synchronize the value of "name" in the form field when the page action changes
    // e.g. from 'view' -> 'clone'
    useEffect(() => {
        const nameValue = {
            create: '',
            view: initialData.name,
            edit: initialData.name,
            clone: `${initialData.name} -COPY-`,
        }[action.type];

        setFieldValue('name', nameValue).catch(() => {
            // Nothing to do on error
        });
    }, [action.type, initialData.name, setFieldValue]);

    const clearConfigError = () => setConfigError(undefined);

    const errors = {
        ...formikErrors,
    };

    // We can associate this type of server error to a specific field, so update the formik errors
    if (configError?.type === 'DuplicateName') {
        errors.name = configError.message;
    }

    if (configError?.type === 'EmptyName') {
        errors.name = configError.message;
    }

    // We only want to display the error in the name field if one of the following is true:
    //   1. The user has focused and blurred the name field, and the value is invalid
    //   2. A request has been sent to the server that resulted in an error, and the name is invalid
    // This prevents an error from being shown as soon as the user loads the creation form, before
    // a name value has been entered.
    const nameError = (touched.name || configError) && errors.name;

    const collectionTableCells = getCollectionTableCells(
        configError?.type === 'CollectionLoop' ? configError.loopId : undefined
    );

    const onResourceSelectorChange = (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector
    ) => setFieldValue(`resourceSelector.${entityType}`, scopedResourceSelector);

    const onEmbeddedCollectionsChange = (newCollections: Collection[]) => {
        if (
            configError?.type === 'CollectionLoop' &&
            !newCollections.find(({ id }) => id === configError.loopId)
        ) {
            clearConfigError();
        }
        return setFieldValue(
            'embeddedCollectionIds',
            newCollections.map(({ id }) => id)
        );
    };

    const ruleCount = getRuleCount(values.resourceSelector);

    return (
        <Form
            className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-h-100"
            style={
                {
                    '--pf-v5-c-form--GridGap': 0,
                } as CSSProperties
            }
        >
            <Flex
                className="pf-v5-u-p-lg pf-v5-u-flex-grow-1 pf-v5-u-background-color-200"
                spaceItems={{ default: 'spaceItemsMd' }}
                direction={{ default: 'column' }}
            >
                <Flex
                    className="pf-v5-u-background-color-100 pf-v5-u-p-lg"
                    direction={{ default: 'column' }}
                    spaceItems={{ default: 'spaceItemsMd' }}
                >
                    <Title headingLevel="h2">Collection details</Title>
                    <Flex direction={{ default: 'column', lg: 'row' }}>
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <FormGroup label="Name" fieldId="name" isRequired={!isReadOnly}>
                                <TextInput
                                    id="name"
                                    name="name"
                                    value={values.name}
                                    validated={nameError ? 'error' : 'default'}
                                    onChange={(e) => {
                                        if (
                                            configError?.type === 'DuplicateName' ||
                                            configError?.type === 'EmptyName'
                                        ) {
                                            clearConfigError();
                                        }
                                        handleChange(e);
                                    }}
                                    onBlur={handleBlur}
                                    readOnlyVariant={isReadOnly ? 'plain' : undefined}
                                />
                                <FormHelperText>
                                    <HelperText>
                                        <HelperTextItem variant={nameError ? 'error' : 'default'}>
                                            {nameError}
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            </FormGroup>
                        </FlexItem>
                        <FlexItem flex={{ default: 'flex_2' }}>
                            <FormGroup label="Description" fieldId="description">
                                <TextInput
                                    id="description"
                                    name="description"
                                    value={values.description}
                                    onChange={(e) => handleChange(e)}
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
                        toggleId={ruleSectionToggleId}
                        isExpanded={isRuleSectionOpen}
                        onToggle={ruleSectionOnToggle}
                    >
                        <Flex
                            alignItems={{ default: 'alignItemsCenter' }}
                            spaceItems={{ default: 'spaceItemsSm' }}
                        >
                            <Title
                                className={isReadOnly ? 'pf-v5-u-mb-0' : 'pf-v5-u-mb-xs'}
                                headingLevel="h2"
                            >
                                Collection rules
                            </Title>
                            <Badge isRead>{ruleCount}</Badge>
                        </Flex>
                        {!isReadOnly && <p>Select deployments using names or labels</p>}
                    </ExpandableSectionToggle>

                    <ExpandableSection
                        isDetached
                        contentId={ruleSectionContentId}
                        toggleId={ruleSectionToggleId}
                        isExpanded={isRuleSectionOpen}
                    >
                        <Flex
                            className="pf-v5-u-p-md"
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsMd' }}
                        >
                            {configError?.type === 'EmptyCollection' && (
                                <Alert
                                    title="At least one rule must be configured or one collection must be attached from the section below"
                                    component="p"
                                    variant="danger"
                                    isInline
                                />
                            )}
                            {configError?.type === 'InvalidRule' && (
                                <Alert
                                    title={configError.message}
                                    component="p"
                                    variant="danger"
                                    isInline
                                >
                                    {configError.details}
                                </Alert>
                            )}
                            <RuleSelector
                                entityType="Deployment"
                                scopedResourceSelector={values.resourceSelector.Deployment}
                                handleChange={onResourceSelectorChange}
                                validationErrors={errors.resourceSelector?.Deployment}
                                isDisabled={isReadOnly}
                            />
                            <Label className="pf-v5-u-px-md pf-v5-u-font-size-md pf-v5-u-align-self-center">
                                in
                            </Label>
                            <RuleSelector
                                entityType="Namespace"
                                scopedResourceSelector={values.resourceSelector.Namespace}
                                handleChange={onResourceSelectorChange}
                                validationErrors={errors.resourceSelector?.Namespace}
                                isDisabled={isReadOnly}
                            />
                            <Label className="pf-v5-u-px-md pf-v5-u-font-size-md pf-v5-u-align-self-center">
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
                    </ExpandableSection>
                </div>

                <div className="collection-form-expandable-section">
                    <ExpandableSectionToggle
                        contentId={attachmentSectionContentId}
                        toggleId={attachmentSectionToggleId}
                        isExpanded={isAttachmentSectionOpen}
                        onToggle={attachmentSectionOnToggle}
                    >
                        <Flex
                            alignItems={{ default: 'alignItemsCenter' }}
                            spaceItems={{ default: 'spaceItemsSm' }}
                        >
                            <Title className="pf-v5-u-mb-xs" headingLevel="h2">
                                Attached collections
                            </Title>
                            <Badge isRead>{values.embeddedCollectionIds.length}</Badge>
                        </Flex>
                        {!isReadOnly && <p>Extend this collection by attaching other sets.</p>}
                    </ExpandableSectionToggle>

                    <ExpandableSection
                        isDetached
                        contentId={attachmentSectionContentId}
                        toggleId={attachmentSectionToggleId}
                        isExpanded={isAttachmentSectionOpen}
                    >
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsMd' }}
                        >
                            {configError?.type === 'EmptyCollection' && (
                                <Alert
                                    title="At least one collection must be attached or one rule must be configured from the section above"
                                    component="p"
                                    variant="danger"
                                    isInline
                                />
                            )}
                            {configError?.type === 'CollectionLoop' && (
                                <Alert
                                    title={configError.message}
                                    component="p"
                                    variant="danger"
                                    isInline
                                >
                                    {configError.details}
                                </Alert>
                            )}
                            {isReadOnly ? (
                                <AttachedCollectionTable
                                    collections={initialEmbeddedCollections}
                                    collectionTableCells={collectionTableCells}
                                />
                            ) : (
                                <div className="pf-v5-u-p-md">
                                    <CollectionAttacher
                                        excludedCollectionId={
                                            action.type === 'edit' ? action.collectionId : null
                                        }
                                        initialEmbeddedCollections={initialEmbeddedCollections}
                                        onSelectionChange={onEmbeddedCollectionsChange}
                                        collectionTableCells={collectionTableCells}
                                    />
                                </div>
                            )}
                        </Flex>
                    </ExpandableSection>
                </div>
            </Flex>
            {action.type !== 'view' && (
                <div className="pf-v5-u-background-color-100 pf-v5-u-p-lg pf-v5-u-py-md">
                    <Button
                        className="pf-v5-u-mr-md"
                        onClick={submitForm}
                        isDisabled={isSubmitting || !!configError || !isValid}
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
