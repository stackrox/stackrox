import React, { ReactElement, useEffect, useState } from 'react';
import { useFormik } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    AlertVariant,
    Button,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Label,
    TextInput,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { OutlinedQuestionCircleIcon } from '@patternfly/react-icons';

import {
    AccessScope,
    EffectiveAccessScopeCluster,
    LabelSelector,
    LabelSelectorsKey,
    computeEffectiveAccessScopeClusters,
} from 'services/AccessScopesService';

import { AccessControlQueryAction } from '../accessControlPaths';

import {
    LabelSelectorsEditingState,
    getIsEditingLabelSelectors,
    getIsValidRules,
    getTemporarilyValidRules,
} from './accessScopes.utils';
import EffectiveAccessScopeTable from './EffectiveAccessScopeTable';
import LabelInclusion from './LabelInclusion';

const labelIconEffectiveAccessScope = (
    <Tooltip
        content="A list of the allowed clusters and namespaces in each cluster that a user role can access. Resources may be included manually or through label selector rules."
        isContentLeftAligned
        maxWidth="24em"
    >
        <div className="pf-c-button pf-m-plain pf-m-small">
            <OutlinedQuestionCircleIcon />
        </div>
    </Tooltip>
);

const labelIconLabelInclusion = (
    <Tooltip
        content={
            <div>
                Configure label selection rules to provide access to clusters and namespaces based
                on their labels
            </div>
        }
        isContentLeftAligned
        maxWidth="24em"
    >
        <div className="pf-c-button pf-m-plain pf-m-small">
            <OutlinedQuestionCircleIcon />
        </div>
    </Tooltip>
);

export type AccessScopeFormProps = {
    isActionable: boolean;
    action?: AccessControlQueryAction;
    accessScope: AccessScope;
    accessScopes: AccessScope[];
    handleCancel: () => void;
    handleEdit: () => void;
    handleSubmit: (values: AccessScope) => Promise<null>; // because the form has only catch and finally
};

function AccessScopeForm({
    isActionable,
    action,
    accessScope,
    accessScopes,
    handleCancel,
    handleEdit,
    handleSubmit,
}: AccessScopeFormProps): ReactElement {
    const [counterComputing, setCounterComputing] = useState(0);
    const [alertCompute, setAlertCompute] = useState<ReactElement | null>(null);
    const [clusters, setClusters] = useState<EffectiveAccessScopeCluster[]>([]);

    const [isSubmitting, setIsSubmitting] = useState(false);
    const [alertSubmit, setAlertSubmit] = useState<ReactElement | null>(null);

    // Disable Save button while editing label selectors.
    const [labelSelectorsEditingState, setLabelSelectorsEditingState] =
        useState<LabelSelectorsEditingState>({
            clusterLabelSelectors: -1,
            namespaceLabelSelectors: -1,
        });

    const { dirty, errors, handleChange, isValid, resetForm, setFieldValue, values } = useFormik({
        initialValues: accessScope,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup
                .string()
                .required()
                .test(
                    'non-unique-name',
                    'Another access scope already has this name',
                    // Return true if current input name is initial name
                    // or no other access scope already has this name.
                    (nameInput) =>
                        nameInput === accessScope.name ||
                        accessScopes.every(({ name }) => nameInput !== name)
                ),
            description: yup.string(),
        }),
    });

    /*
     * A label selector or set requirement is temporarily invalid when it is added,
     * before its first requirement or value has been added.
     */
    const isValidRules = getIsValidRules(values.rules);

    useEffect(() => {
        setCounterComputing((counterPrev) => counterPrev + 1);
        computeEffectiveAccessScopeClusters(
            isValidRules ? values.rules : getTemporarilyValidRules(values.rules)
        )
            .then((clustersComputed) => {
                setClusters(clustersComputed);
                setAlertCompute(null);
            })
            .catch((error) => {
                setAlertCompute(
                    <Alert
                        title="Compute effective access scope failed"
                        variant={AlertVariant.danger}
                        isInline
                    >
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setCounterComputing((counterPrev) => counterPrev - 1);
            });
    }, [isValidRules, values.rules]);

    function onChange(_value, event) {
        handleChange(event);
    }

    function handleIncludedClustersChange(clusterName: string, isChecked: boolean) {
        const { includedClusters } = values.rules;
        return setFieldValue(
            'rules.includedClusters',
            isChecked
                ? [...includedClusters, clusterName]
                : includedClusters.filter(
                      (includedClusterName) => includedClusterName !== clusterName
                  )
        );
    }

    function handleIncludedNamespacesChange(
        clusterName: string,
        namespaceName: string,
        isChecked: boolean
    ) {
        const { includedNamespaces } = values.rules;
        return setFieldValue(
            'rules.includedNamespaces',
            isChecked
                ? [...includedNamespaces, { clusterName, namespaceName }]
                : includedNamespaces.filter(
                      ({
                          clusterName: includedClusterName,
                          namespaceName: includedNamespaceName,
                      }) =>
                          includedClusterName !== clusterName ||
                          includedNamespaceName !== namespaceName
                  )
        );
    }

    function handleLabelSelectorsChange(
        labelSelectorsKey: LabelSelectorsKey,
        labelSelectors: LabelSelector[]
    ) {
        return setFieldValue(`rules.${labelSelectorsKey}`, labelSelectors);
    }

    function onClickSubmit() {
        // TODO submit through Formik, especially to update its initialValue.
        // For example, to make a change, submit, and then make the opposite change.
        setIsSubmitting(true);
        setAlertSubmit(null);
        handleSubmit(values)
            .catch((error) => {
                setAlertSubmit(
                    <Alert
                        title="Failed to save access scope"
                        variant={AlertVariant.danger}
                        isInline
                    >
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setIsSubmitting(false);
                resetForm({ values });
            });
    }

    function onClickCancel() {
        resetForm();
        handleCancel(); // close form if action=create but not if action=update
    }

    const hasAction = Boolean(action);
    const isViewing = !hasAction;

    const nameErrorMessage = values.name.length !== 0 && errors.name ? errors.name : '';
    const nameValidatedState = nameErrorMessage ? 'error' : 'default';

    return (
        <Form id="access-scope-form">
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarItem>
                        <Title headingLevel="h2">
                            {action === 'create' ? 'Add access scope' : accessScope.name}
                        </Title>
                    </ToolbarItem>
                    {action !== 'create' && (
                        <ToolbarGroup variant="button-group" alignment={{ default: 'alignRight' }}>
                            <ToolbarItem>
                                {isActionable ? (
                                    <Button
                                        variant="primary"
                                        onClick={handleEdit}
                                        isDisabled={action === 'edit'}
                                        isSmall
                                    >
                                        Edit access scope
                                    </Button>
                                ) : (
                                    <Label>Not editable</Label>
                                )}
                            </ToolbarItem>
                        </ToolbarGroup>
                    )}
                </ToolbarContent>
            </Toolbar>
            {alertSubmit}
            <FormGroup
                label="Name"
                fieldId="name"
                isRequired
                validated={nameValidatedState}
                helperTextInvalid={nameErrorMessage}
                className="pf-m-horizontal"
            >
                <TextInput
                    type="text"
                    id="name"
                    value={values.name}
                    validated={nameValidatedState}
                    onChange={onChange}
                    isDisabled={isViewing}
                    isRequired
                    className="pf-m-limit-width"
                />
            </FormGroup>
            <FormGroup label="Description" fieldId="description" className="pf-m-horizontal">
                <TextInput
                    type="text"
                    id="description"
                    value={values.description}
                    onChange={onChange}
                    isDisabled={isViewing}
                />
            </FormGroup>
            {alertCompute}
            <Flex
                direction={{ default: 'row' }}
                spaceItems={{ default: 'spaceItemsSm', xl: 'spaceItemsLg' }}
            >
                <FlexItem className="pf-u-flex-basis-0" flex={{ default: 'flex_1' }}>
                    <FormGroup
                        label="Allowed resources"
                        fieldId="effectiveAccessScope"
                        labelIcon={labelIconEffectiveAccessScope}
                    >
                        <EffectiveAccessScopeTable
                            counterComputing={counterComputing}
                            clusters={clusters}
                            includedClusters={values.rules.includedClusters}
                            includedNamespaces={values.rules.includedNamespaces}
                            handleIncludedClustersChange={handleIncludedClustersChange}
                            handleIncludedNamespacesChange={handleIncludedNamespacesChange}
                            hasAction={hasAction}
                        />
                    </FormGroup>
                </FlexItem>
                <FlexItem className="pf-u-flex-basis-0" flex={{ default: 'flex_1' }}>
                    <FormGroup
                        label="Label selection rules"
                        fieldId="labelInclusion"
                        labelIcon={labelIconLabelInclusion}
                    >
                        <LabelInclusion
                            clusterLabelSelectors={values.rules.clusterLabelSelectors}
                            namespaceLabelSelectors={values.rules.namespaceLabelSelectors}
                            hasAction={hasAction}
                            labelSelectorsEditingState={labelSelectorsEditingState}
                            setLabelSelectorsEditingState={setLabelSelectorsEditingState}
                            handleLabelSelectorsChange={handleLabelSelectorsChange}
                        />
                    </FormGroup>
                </FlexItem>
            </Flex>
            {hasAction && (
                <Toolbar inset={{ default: 'insetNone' }} className="pf-u-pb-0">
                    <ToolbarContent>
                        <ToolbarGroup variant="button-group">
                            <ToolbarItem>
                                <Button
                                    variant="primary"
                                    onClick={onClickSubmit}
                                    isDisabled={
                                        !dirty ||
                                        !isValid ||
                                        !isValidRules ||
                                        getIsEditingLabelSelectors(labelSelectorsEditingState) ||
                                        isSubmitting
                                    }
                                    isLoading={isSubmitting}
                                    isSmall
                                >
                                    Save
                                </Button>
                            </ToolbarItem>
                            <ToolbarItem>
                                <Button variant="tertiary" onClick={onClickCancel} isSmall>
                                    Cancel
                                </Button>
                            </ToolbarItem>
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
            )}
        </Form>
    );
}

export default AccessScopeForm;
