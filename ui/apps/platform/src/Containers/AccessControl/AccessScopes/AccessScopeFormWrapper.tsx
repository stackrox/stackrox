import React, { ReactElement, useState } from 'react';
import { useFormik } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    AlertVariant,
    Button,
    Label,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import { AccessScope, getIsUnrestrictedAccessScopeId } from 'services/AccessScopesService';

import { AccessControlQueryAction } from '../accessControlPaths';

import {
    LabelSelectorsEditingState,
    getIsEditingLabelSelectors,
    getIsValidRules,
} from './accessScopes.utils';
import AccessScopeForm from './AccessScopeForm';
import usePermissions from '../../../hooks/usePermissions';
import { TraitsOriginLabel } from '../TraitsOriginLabel';

export type AccessScopeFormWrapperProps = {
    isActionable: boolean;
    action?: AccessControlQueryAction;
    accessScope: AccessScope;
    accessScopes: AccessScope[];
    handleCancel: () => void;
    handleEdit: () => void;
    handleSubmit: (values: AccessScope) => Promise<null>; // because the form has only catch and finally
};

function AccessScopeFormWrapper({
    isActionable,
    action,
    accessScope,
    accessScopes,
    handleCancel,
    handleEdit,
    handleSubmit,
}: AccessScopeFormWrapperProps): ReactElement {
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [alertSubmit, setAlertSubmit] = useState<ReactElement | null>(null);
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForPage = hasReadWriteAccess('Access');

    // Disable Save button while editing label selectors.
    const [labelSelectorsEditingState, setLabelSelectorsEditingState] =
        useState<LabelSelectorsEditingState>({
            clusterLabelSelectors: -1,
            namespaceLabelSelectors: -1,
        });

    const formik = useFormik({
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
    const { dirty, isValid, resetForm, values } = formik;

    /*
     * A label selector or set requirement is temporarily invalid when it is added,
     * before its first requirement or value has been added.
     */
    const isValidRules =
        !getIsUnrestrictedAccessScopeId(values.id) && getIsValidRules(values.rules);

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

    return (
        <>
            <Toolbar inset={{ default: 'insetNone' }} className="pf-u-pt-0">
                <ToolbarContent>
                    <ToolbarItem>
                        <Title headingLevel="h2">
                            {action === 'create' ? 'Create access scope' : accessScope.name}
                        </Title>
                    </ToolbarItem>
                    {action !== 'create' && (
                        <ToolbarItem>
                            <TraitsOriginLabel traits={accessScope.traits} />
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
            <AccessScopeForm
                hasAction={hasAction}
                alertSubmit={alertSubmit}
                formik={formik}
                labelSelectorsEditingState={labelSelectorsEditingState}
                setLabelSelectorsEditingState={setLabelSelectorsEditingState}
            />
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
        </>
    );
}

export default AccessScopeFormWrapper;
