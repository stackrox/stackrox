import React, { ReactElement, useState } from 'react';
import { useFormik } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    AlertVariant,
    Badge,
    Button,
    Form,
    FormGroup,
    Label,
    TextInput,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import { defaultMinimalReadAccessResources } from 'constants/accessControl';
import { PermissionSet } from 'services/RolesService';

import { AccessControlQueryAction } from '../accessControlPaths';

import PermissionsTable from './PermissionsTable';
import usePermissions from '../../../hooks/usePermissions';
import { TraitsOriginLabel } from '../TraitsOriginLabel';

export type PermissionSetFormProps = {
    isActionable: boolean;
    action?: AccessControlQueryAction;
    permissionSet: PermissionSet;
    permissionSets: PermissionSet[];
    handleCancel: () => void;
    handleEdit: () => void;
    handleSubmit: (values: PermissionSet) => Promise<null>; // because the form has only catch and finally
};

function PermissionSetForm({
    isActionable,
    action,
    permissionSet,
    permissionSets,
    handleCancel,
    handleEdit,
    handleSubmit,
}: PermissionSetFormProps): ReactElement {
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [alertSubmit, setAlertSubmit] = useState<ReactElement | null>(null);
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForPage = hasReadWriteAccess('Access');

    const { dirty, errors, handleChange, isValid, resetForm, setFieldValue, values } = useFormik({
        initialValues: permissionSet,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup
                .string()
                .required()
                .test(
                    'non-unique-name',
                    'Another permission set already has this name',
                    // Return true if current input name is initial name
                    // or no other permission set already has this name.
                    (nameInput) =>
                        nameInput === permissionSet.name ||
                        permissionSets.every(({ name }) => nameInput !== name)
                ),
            description: yup.string(),
            // resourceToAccess is valid because selections of access level
        }),
    });

    function setResourceValue(resource, value) {
        const { resourceToAccess } = values;
        return setFieldValue('resourceToAccess', {
            ...resourceToAccess,
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
        handleSubmit(values)
            .catch((error) => {
                setAlertSubmit(
                    <Alert
                        title="Failed to save permission set"
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

    const nameErrorMessage = values.name.length !== 0 && errors.name ? errors.name : '';
    const nameValidatedState = nameErrorMessage ? 'error' : 'default';

    return (
        <Form id="permission-set-form">
            <Toolbar inset={{ default: 'insetNone' }} className="pf-u-pt-0">
                <ToolbarContent>
                    <ToolbarItem>
                        <Title headingLevel="h2">
                            {action === 'create' ? 'Create permission set' : permissionSet.name}
                        </Title>
                    </ToolbarItem>
                    {action !== 'create' && (
                        <ToolbarItem>
                            <TraitsOriginLabel traits={permissionSet.traits} />
                        </ToolbarItem>
                    )}
                    {action !== 'create' && (
                        <ToolbarGroup variant="button-group" alignment={{ default: 'alignRight' }}>
                            <ToolbarItem>
                                {isActionable ? (
                                    <Button
                                        variant="primary"
                                        onClick={handleEdit}
                                        isDisabled={!hasWriteAccessForPage || action === 'edit'}
                                        isSmall
                                    >
                                        Edit permission set
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
            {action === 'create' && (
                <Alert title="Recommended minimum set of read permissions" variant="info" isInline>
                    <p>
                        Users might not be able to load certain pages if they do not have a minimum
                        set of read permissions.
                    </p>
                    <br />
                    <p>
                        If this permission set is for <strong>users</strong>, select at least{' '}
                        <strong>Read access</strong> for at least the following resources:
                    </p>
                    <p>
                        <strong>{defaultMinimalReadAccessResources.join(', ')}</strong>
                        <Badge isRead className="pf-u-ml-sm">
                            {defaultMinimalReadAccessResources.length}
                        </Badge>
                    </p>
                </Alert>
            )}
            <FormGroup label="Permissions" fieldId="permissions" isRequired>
                <PermissionsTable
                    resourceToAccess={values.resourceToAccess}
                    setResourceValue={setResourceValue}
                    isDisabled={isViewing}
                />
            </FormGroup>
            {hasAction && (
                <Toolbar inset={{ default: 'insetNone' }}>
                    <ToolbarContent>
                        <ToolbarGroup variant="button-group">
                            <ToolbarItem>
                                <Button
                                    variant="primary"
                                    onClick={onClickSubmit}
                                    isDisabled={!dirty || !isValid || isSubmitting}
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

export default PermissionSetForm;
