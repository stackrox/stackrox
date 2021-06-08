import React, { ReactElement, useState } from 'react';
import { useFormik } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    AlertVariant,
    Button,
    Form,
    FormGroup,
    SelectOption,
    TextInput,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import { accessControl as accessControlTypeLabels } from 'messages/common';

import { AccessControlQueryAction, PermissionSet } from '../accessControlTypes';
import ResourcesTable from './ResourcesTable';
import SelectSingle from '../SelectSingle'; // TODO import from where?

export type PermissionSetFormProps = {
    isActionable: boolean;
    action?: AccessControlQueryAction;
    permissionSet: PermissionSet;
    onClickCancel: () => void;
    onClickEdit: () => void;
    submitValues: (values: PermissionSet) => Promise<PermissionSet>;
};

function PermissionSetForm({
    isActionable,
    action,
    permissionSet,
    onClickCancel,
    onClickEdit,
    submitValues,
}: PermissionSetFormProps): ReactElement {
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [alertSubmit, setAlertSubmit] = useState<ReactElement | null>(null);

    // TODO Why does browser refresh when form is open cause values to be undefined?
    const { dirty, handleChange, isValid, setFieldValue, values } = useFormik({
        initialValues: permissionSet,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup.string().required(),
            description: yup.string(),
            // minimumAccessLevel
            // permissions
        }),
    });

    function setResourceValue(resource, value) {
        const { resourceIdToAccess } = values;
        return setFieldValue('resourceIdToAccess', {
            ...resourceIdToAccess,
            [resource]: value,
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
        submitValues(values)
            .catch((error) => {
                setAlertSubmit(
                    <Alert
                        title="Failed to submit permission set"
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

    // TODO Miminum access level: does not need full width.
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
                                    Edit permission set
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
            <FormGroup label="Minimum access level" fieldId="minimumAccessLevel" isRequired>
                <SelectSingle
                    id="minimumAccessLevel"
                    value={values.minimumAccessLevel}
                    setFieldValue={setFieldValue}
                    isDisabled={isViewing}
                >
                    {Object.entries(accessControlTypeLabels).map(([value, label]) => (
                        <SelectOption key={value} value={value}>
                            {label}
                        </SelectOption>
                    ))}
                </SelectSingle>
            </FormGroup>
            <ResourcesTable
                resourceIdToAccess={values.resourceIdToAccess}
                setResourceValue={setResourceValue}
                isDisabled={isViewing}
            />
        </Form>
    );
}

export default PermissionSetForm;
