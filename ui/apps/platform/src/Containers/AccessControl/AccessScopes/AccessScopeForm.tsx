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
    TextInput,
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
    computeEffectiveAccessScopeClusters,
} from 'services/RolesService';

import { AccessControlQueryAction } from '../accessControlPaths';

import EffectiveAccessScopeTable from './EffectiveAccessScopeTable';
import LabelInclusion from './LabelInclusion';

// Form group label icon style rule in AccessScopes.css mimics info prop in table head cells.

const labelIconEffectiveAccessScope = (
    <Tooltip
        content={
            <div>
                Status of clusters and namespaces
                <br />
                from manual inclusion, or label inclusion, or both
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

const labelIconLabelInclusion = (
    <Tooltip
        content={
            <div>
                You can specify label selectors
                <br />
                in addition to, or instead of, manual inclusion
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
    handleCancel: () => void;
    handleEdit: () => void;
    handleSubmit: (values: AccessScope) => Promise<null>; // because the form has only catch and finally
};

function AccessScopeForm({
    isActionable,
    action,
    accessScope,
    handleCancel,
    handleEdit,
    handleSubmit,
}: AccessScopeFormProps): ReactElement {
    const [counterComputing, setCounterComputing] = useState(0);
    const [clusters, setClusters] = useState<EffectiveAccessScopeCluster[]>([]);

    const [isSubmitting, setIsSubmitting] = useState(false);
    const [alertSubmit, setAlertSubmit] = useState<ReactElement | null>(null);

    const { dirty, handleChange, isValid, resetForm, setFieldValue, values } = useFormik({
        initialValues: accessScope,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup.string().required(),
            description: yup.string(),
        }),
    });

    useEffect(() => {
        setCounterComputing((counterPrev) => counterPrev + 1);
        computeEffectiveAccessScopeClusters(values.rules)
            .then((clustersComputed) => {
                setClusters(clustersComputed);
            })
            .catch(() => {
                // TODO display alert
            })
            .finally(() => {
                setCounterComputing((counterPrev) => counterPrev - 1);
            });
    }, [values.rules]);

    function handleChangeIncludedClusters(clusterName: string, isChecked: boolean) {
        const {
            includedClusters,
            includedNamespaces,
            clusterLabelSelectors,
            namespaceLabelSelectors,
        } = values.rules;
        return setFieldValue('rules', {
            includedClusters: isChecked
                ? [...includedClusters, clusterName]
                : includedClusters.filter(
                      (includedClusterName) => includedClusterName !== clusterName
                  ),
            includedNamespaces,
            clusterLabelSelectors,
            namespaceLabelSelectors,
        });
    }

    function handleChangeIncludedNamespaces(
        clusterName: string,
        namespaceName: string,
        isChecked: boolean
    ) {
        const {
            includedClusters,
            includedNamespaces,
            clusterLabelSelectors,
            namespaceLabelSelectors,
        } = values.rules;
        return setFieldValue('rules', {
            includedClusters,
            includedNamespaces: isChecked
                ? [...includedNamespaces, { clusterName, namespaceName }]
                : includedNamespaces.filter(
                      ({
                          clusterName: includedClusterName,
                          namespaceName: includedNamespaceName,
                      }) =>
                          includedClusterName !== clusterName ||
                          includedNamespaceName !== namespaceName
                  ),
            clusterLabelSelectors,
            namespaceLabelSelectors,
        });
    }

    function onChange(_value, event) {
        handleChange(event);
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
                        title="Failed to submit access scope"
                        variant={AlertVariant.danger}
                        isInline
                    >
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setIsSubmitting(false);
            });
    }

    function onClickCancel() {
        resetForm();
        handleCancel(); // close form if action=create but not if action=update
    }

    const hasAction = Boolean(action);
    const isViewing = !hasAction;

    return (
        <Form id="access-scope-form">
            {isActionable && (
                <Toolbar inset={{ default: 'insetNone' }}>
                    <ToolbarContent>
                        {action !== 'create' && (
                            <ToolbarItem spacer={{ default: 'spacerLg' }}>
                                <Button
                                    variant="primary"
                                    onClick={handleEdit}
                                    isDisabled={action === 'update'}
                                    isSmall
                                >
                                    Edit access scope
                                </Button>
                            </ToolbarItem>
                        )}
                        {hasAction && (
                            <ToolbarGroup variant="button-group">
                                <ToolbarItem>
                                    <Button
                                        variant="primary"
                                        onClick={onClickSubmit}
                                        isDisabled={!dirty || !isValid || isSubmitting}
                                        isLoading={isSubmitting}
                                        isSmall
                                    >
                                        Submit
                                    </Button>
                                </ToolbarItem>
                                <ToolbarItem>
                                    <Button variant="tertiary" onClick={onClickCancel} isSmall>
                                        Cancel
                                    </Button>
                                </ToolbarItem>
                            </ToolbarGroup>
                        )}
                    </ToolbarContent>
                </Toolbar>
            )}
            {alertSubmit}
            <FormGroup label="Name" fieldId="name" isRequired>
                <TextInput
                    type="text"
                    id="name"
                    value={values.name}
                    onChange={onChange}
                    isDisabled={isViewing}
                    isRequired
                />
            </FormGroup>
            <FormGroup label="Description" fieldId="description">
                <TextInput
                    type="text"
                    id="description"
                    value={values.description}
                    onChange={onChange}
                    isDisabled={isViewing}
                />
            </FormGroup>
            <Flex>
                <FlexItem className="pf-u-flex-basis-0 pf-u-flex-grow-1">
                    <FormGroup
                        label="Effective access scope"
                        fieldId="effectiveAccessScope"
                        labelIcon={labelIconEffectiveAccessScope}
                    >
                        <EffectiveAccessScopeTable
                            counterComputing={counterComputing}
                            clusters={clusters}
                            includedClusters={values.rules.includedClusters}
                            includedNamespaces={values.rules.includedNamespaces}
                            handleChangeIncludedClusters={handleChangeIncludedClusters}
                            handleChangeIncludedNamespaces={handleChangeIncludedNamespaces}
                            hasAction={hasAction}
                        />
                    </FormGroup>
                </FlexItem>
                <FlexItem className="pf-u-flex-basis-0 pf-u-flex-grow-1">
                    <FormGroup
                        label="Label inclusion"
                        fieldId="labelInclusion"
                        labelIcon={labelIconLabelInclusion}
                    >
                        <LabelInclusion
                            clusterLabelSelectors={values.rules.clusterLabelSelectors}
                            namespaceLabelSelectors={values.rules.namespaceLabelSelectors}
                            hasAction={hasAction}
                        />
                    </FormGroup>
                </FlexItem>
            </Flex>
        </Form>
    );
}

export default AccessScopeForm;
