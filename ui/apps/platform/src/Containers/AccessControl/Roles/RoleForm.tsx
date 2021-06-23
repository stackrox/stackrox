import React, { ReactElement, useState } from 'react';
import { useFormik } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    AlertVariant,
    Button,
    Form,
    FormGroup,
    TextInput,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import { AccessScope, PermissionSet, Role } from 'services/RolesService';

import { AccessControlQueryAction } from '../accessControlPaths';

import AccessScopesTable from './AccessScopesTable';
import PermissionSetsTable from './PermissionSetsTable';

export type RoleFormProps = {
    isActionable: boolean;
    action?: AccessControlQueryAction;
    role: Role;
    permissionSets: PermissionSet[];
    accessScopes: AccessScope[];
    handleCancel: () => void;
    handleEdit: () => void;
    handleSubmit: (values: Role) => Promise<null>; // because the form has only catch and finally
};

function RoleForm({
    isActionable,
    action,
    role,
    permissionSets,
    accessScopes,
    handleCancel,
    handleEdit,
    handleSubmit,
}: RoleFormProps): ReactElement {
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [alertSubmit, setAlertSubmit] = useState<ReactElement | null>(null);

    const { dirty, handleChange, isValid, values } = useFormik({
        initialValues: role,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup.string().required(),
            description: yup.string(),
            permissionSetId: yup.string().required(),
            accessScopeId: yup.string(),
        }),
    });

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
                    <Alert title="Failed to submit role" variant={AlertVariant.danger} isInline>
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
        <Form id="role-form">
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarItem>
                        <Title headingLevel="h2">
                            {action === 'create' ? 'Create role' : role.name}
                        </Title>
                    </ToolbarItem>
                    {isActionable && (
                        <ToolbarGroup
                            alignment={{ default: 'alignRight' }}
                            spaceItems={{ default: 'spaceItemsLg' }}
                        >
                            {hasAction ? (
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
                                        <Button variant="tertiary" onClick={handleCancel} isSmall>
                                            Cancel
                                        </Button>
                                    </ToolbarItem>
                                </ToolbarGroup>
                            ) : (
                                <ToolbarItem>
                                    <Button
                                        variant="primary"
                                        onClick={handleEdit}
                                        isDisabled={action === 'update'}
                                        isSmall
                                    >
                                        Edit role
                                    </Button>
                                </ToolbarItem>
                            )}
                        </ToolbarGroup>
                    )}
                </ToolbarContent>
            </Toolbar>
            {alertSubmit}
            <FormGroup label="Name" fieldId="name" isRequired className="pf-m-horizontal">
                <TextInput
                    type="text"
                    id="name"
                    value={values.name}
                    onChange={onChange}
                    isDisabled={isViewing}
                    isRequired
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
            <FormGroup label="Permission set" fieldId="permissionSetId" isRequired>
                <PermissionSetsTable
                    fieldId="permissionSetId"
                    permissionSetId={values.permissionSetId}
                    permissionSets={permissionSets}
                    handleChange={handleChange}
                    isDisabled={isViewing}
                />
            </FormGroup>
            <FormGroup label="Access scope" fieldId="accessScopeId">
                {accessScopes.length !== 0 && (
                    <AccessScopesTable
                        fieldId="accessScopeId"
                        accessScopeId={values.accessScopeId}
                        accessScopes={accessScopes}
                        handleChange={handleChange}
                        isDisabled={isViewing}
                    />
                )}
            </FormGroup>
        </Form>
    );
}

export default RoleForm;
