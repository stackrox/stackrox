import React, { ReactElement, useEffect, useState } from 'react';
import { useFormik } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    AlertVariant,
    Button,
    Form,
    FormGroup,
    TextInput,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import {
    AccessScope,
    EffectiveAccessScopeCluster,
    computeEffectiveAccessScopeClusters,
} from 'services/RolesService';

import { AccessControlQueryAction } from '../accessControlPaths';

export type AccessScopeFormProps = {
    isActionable: boolean;
    action?: AccessControlQueryAction;
    accessScope: AccessScope;
    onClickCancel: () => void;
    onClickEdit: () => void;
    submitValues: (values: AccessScope) => Promise<null>; // because the form has only catch and finally
};

function AccessScopeForm({
    isActionable,
    action,
    accessScope,
    onClickCancel,
    onClickEdit,
    submitValues,
}: AccessScopeFormProps): ReactElement {
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [alertSubmit, setAlertSubmit] = useState<ReactElement | null>(null);
    const [clusters, setClusters] = useState<EffectiveAccessScopeCluster[]>([]);

    const { dirty, handleChange, isValid, values } = useFormik({
        initialValues: accessScope,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup.string().required(),
            description: yup.string(),
        }),
    });

    useEffect(() => {
        // TODO set computing true
        computeEffectiveAccessScopeClusters(accessScope.rules)
            .then((clustersArg) => {
                setClusters(clustersArg);
            })
            .catch(() => {
                // TODO display alert
            })
            .finally(() => {
                // TODO set computing false
            });
    }, [accessScope.rules]);

    function onChange(_value, event) {
        handleChange(event);
    }

    function onClickSubmit() {
        // TODO submit through Formik, especially to update its initialValue.
        // For example, to make a change, submit, and then make the opposite change.
        setIsSubmitting(true);
        setAlertSubmit(null);
        submitValues(values)
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

    const hasAction = Boolean(action);
    const isViewing = !hasAction;

    return (
        <Form>
            {isActionable && (
                <Toolbar inset={{ default: 'insetNone' }}>
                    <ToolbarContent>
                        {action !== 'create' && (
                            <ToolbarItem spacer={{ default: 'spacerLg' }}>
                                <Button
                                    variant="primary"
                                    onClick={onClickEdit}
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
            <FormGroup label="Effective Access Scope" fieldId="effectiveAccessScope">
                <pre className="pf-u-font-size-sm">{JSON.stringify(clusters, null, 2)}</pre>
            </FormGroup>
        </Form>
    );
}

export default AccessScopeForm;
