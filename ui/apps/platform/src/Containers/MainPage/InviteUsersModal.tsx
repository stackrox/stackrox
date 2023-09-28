import React, { useEffect, ReactElement } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Modal, ModalVariant, ModalBoxBody, ModalBoxFooter, Button } from '@patternfly/react-core';
import { useFormik } from 'formik';
import * as yup from 'yup';

import { selectors } from 'reducers';
import { actions as inviteActions } from 'reducers/invite';
import { actions as authActions, types as authActionTypes } from 'reducers/auth';
import { actions as groupActions } from 'reducers/groups';
import { actions as roleActions, types as roleActionTypes } from 'reducers/roles';
import { AuthProvider } from 'services/AuthService/AuthService';
import { dedupeDelimtedString } from 'utils/textUtils';
import { mergeGroupsWithAuthProviders } from '../AccessControl/AuthProviders/authProviders.utils';
import InviteUsersForm from './InviteUsersForm';

type InviteFormValues = {
    emails: string;
    provider: string;
    role: string;
};

// email validation from discussion in Yup repo,
// https://github.com/jquense/yup/issues/564#issuecomment-536068508
const isEmail = (value) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
const validationSchema = yup.object().shape({
    emails: yup
        .string()
        .transform((value) => dedupeDelimtedString(value).join(',')) // dedupe - optional step
        .required('At least one email is required')
        .test('emails', 'Invalid email address', (value) =>
            Boolean(
                value &&
                    value
                        .split(',')
                        .map((v) => v.trim())
                        .every(isEmail)
            )
        ), // .string().trim().required('At least one email is required'),
    provider: yup.string().required('An auth provider is required'),
    role: yup.string().required('A role is required'),
});

const defaultInviteFormValues: InviteFormValues = {
    emails: '',
    provider: '',
    role: '',
};

const feedbackState = createStructuredSelector({
    authProviders: selectors.getAvailableAuthProviders,
    groups: selectors.getRuleGroups,
    roles: selectors.getRoles,
    isFetchingAuthProviders: (state) =>
        selectors.getLoadingStatus(state, authActionTypes.FETCH_AUTH_PROVIDERS) as boolean,
    isFetchingRoles: (state) =>
        selectors.getLoadingStatus(state, roleActionTypes.FETCH_ROLES) as boolean,
    invite: selectors.inviteSelector,
});

function InviteUsersModal(): ReactElement | null {
    const { invite: showInviteModal } = useSelector(feedbackState);
    const dispatch = useDispatch();
    const formik = useFormik<InviteFormValues>({
        initialValues: defaultInviteFormValues,
        onSubmit: () => {}, // required but not used, because the submit action is in the modal footer
        validationSchema,
        validateOnMount: true,
    });

    const { isValid, values, setFieldValue } = formik;

    const { authProviders, groups, roles } = useSelector(feedbackState);

    const authProvidersWithRules = mergeGroupsWithAuthProviders(authProviders, groups);

    const allowedRoles = roles.filter((role) => {
        return role.name !== 'Admin' && role.name !== 'None';
    });

    useEffect(() => {
        dispatch(authActions.fetchAuthProviders.request());
        dispatch(roleActions.fetchRoles.request());
        dispatch(groupActions.fetchGroups.request());
    }, [dispatch]);

    useEffect(() => {
        const typedProviders = authProviders as AuthProvider[]; // redux state from sagas is not yet typed

        // if there is only 1 authProvider, pre-select it in the dropdown
        if (typedProviders.length === 1) {
            // eslint-disable-next-line no-void
            void setFieldValue('provider', typedProviders[0].name);
        } else if (typedProviders.length > 1) {
            // if there is more than 1 authProvider, pre-select RedHat / OpenShift provider if possible
            const redhatSsoProvider = typedProviders.find(
                (provider) =>
                    provider.name.includes('Red Hat SSO') || provider.name.includes('Red Hat SSO')
            );

            if (redhatSsoProvider) {
                // eslint-disable-next-line no-void
                void setFieldValue('provider', redhatSsoProvider.name);
            }
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [authProviders]); // ignore `setFieldValue` as a dependency

    function onChange(value, event) {
        // handle PF4 inconsistency with Select inputs, where value is ID, and event is Value
        if (value === 'role' || value === 'provider') {
            return setFieldValue(value, event);
        }
        return setFieldValue(event.target.id, value);
    }

    function submitInvitations() {
        console.log({ values });
    }

    function onClose() {
        dispatch(inviteActions.setInviteModalVisibility(false));
    }

    return (
        <Modal
            title="Invite users"
            isOpen={showInviteModal}
            variant={ModalVariant.small}
            onClose={onClose}
            aria-label="Permanently delete category?"
            hasNoBodyWrapper
        >
            <ModalBoxBody>
                {/* <p>{JSON.stringify(authProvidersWithRules)}</p> */}
                <InviteUsersForm
                    formik={formik}
                    providers={authProviders}
                    roles={allowedRoles}
                    onChange={onChange}
                />
            </ModalBoxBody>
            <ModalBoxFooter>
                <Button
                    key="invite"
                    variant="primary"
                    onClick={submitInvitations}
                    isDisabled={!isValid}
                >
                    Invite users
                </Button>
                <Button key="cancel" variant="link" onClick={onClose}>
                    Cancel
                </Button>
            </ModalBoxFooter>
        </Modal>
    );
}

export default InviteUsersModal;
