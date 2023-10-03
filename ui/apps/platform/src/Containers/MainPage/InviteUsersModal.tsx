import React, { useEffect, ReactElement } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import {
    Alert,
    Button,
    Modal,
    ModalVariant,
    ModalBoxBody,
    ModalBoxFooter,
    Text,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom';
import { useFormik } from 'formik';
import * as yup from 'yup';

import { selectors } from 'reducers';
import { actions as inviteActions } from 'reducers/invite';
import { actions as authActions } from 'reducers/auth';
import { actions as groupActions } from 'reducers/groups';
import { actions as roleActions } from 'reducers/roles';
import { accessControlBasePath } from 'routePaths';
import { AuthProvider } from 'services/AuthService';
import { dedupeDelimitedString } from 'utils/textUtils';
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
        .transform((value) => dedupeDelimitedString(value).join(',')) // dedupe - optional step
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
    showInviteModal: selectors.inviteSelector,
});

function InviteUsersModal(): ReactElement | null {
    const { authProviders, groups, roles, showInviteModal } = useSelector(feedbackState);
    const authProvidersWithRules = mergeGroupsWithAuthProviders(authProviders, groups);

    const dispatch = useDispatch();

    const formik = useFormik<InviteFormValues>({
        initialValues: defaultInviteFormValues,
        onSubmit: () => {}, // required but not used, because the submit action is in the modal footer
        validationSchema,
        validateOnMount: true,
    });

    const { isValid, values, setFieldValue } = formik;

    const allowedRoles = roles.filter((role) => {
        return role.name !== 'Admin' && role.name !== 'None';
    });

    useEffect(() => {
        dispatch(authActions.fetchAuthProviders.request());
        dispatch(roleActions.fetchRoles.request());
        dispatch(groupActions.fetchGroups.request());
    }, [dispatch]);

    useEffect(() => {
        // redux state from sagas is not yet typed, and API response could corrupt Redux at runtime
        if (!Array.isArray(authProviders)) {
            return;
        }
        const typedProviders = (authProviders as AuthProvider[]).filter(
            (provider) => typeof provider.name === 'string'
        );

        // if there is only 1 authProvider, pre-select it in the dropdown
        if (typedProviders.length === 1) {
            // eslint-disable-next-line no-void
            void setFieldValue('provider', typedProviders[0].name);
        } else if (typedProviders.length > 1) {
            // if there is more than 1 authProvider, pre-select RedHat / OpenShift provider if possible
            const redhatSsoProvider = typedProviders.find((provider) => {
                const lowercasedName = provider.name.toLocaleLowerCase();
                return (
                    lowercasedName.includes('red Hat sso') || lowercasedName.includes('openshift')
                );
            });

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
        // eslint-disable-next-line no-console
        console.log({ values });

        // check whether any of the listed emails already have rules for this auth provider
        const providerWithRules = authProvidersWithRules.find(
            (provider) => provider.name === values.provider
        );
        if (providerWithRules) {
            const emailArr = dedupeDelimitedString(values.emails);
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
            const emailBuckets = emailArr.reduce<{ new: string[]; existing: string[] }>(
                (acc, email) => {
                    if (
                        providerWithRules.groups?.some(
                            (group) => group.props.key === 'email' && group.props.value === email
                        )
                    ) {
                        return { new: acc.new, existing: [...acc.existing, email] };
                    }
                    return { new: [...acc.new, email], existing: acc.existing };
                },
                { new: [], existing: [] }
            );

            // TODO: Show warning if some emails will not be added
        }

        // TODO: detect if an email server is available
        //       1. if so, send emails to the listed recipients
        //       2. if not, show the emails and the message body for copying to a manual email
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
                {authProviders.length === 0 && (
                    <Alert
                        title="No auth providers are available."
                        variant="warning"
                        isInline
                        className="pf-u-mb-lg"
                    >
                        <Text>
                            You must have at least one auth provider in order to invite users.
                        </Text>
                        <Text>
                            Visit the{' '}
                            <Link onClick={onClose} to={`${accessControlBasePath}/auth-providers`}>
                                Access Control
                            </Link>{' '}
                            section to add an auth provider.
                        </Text>
                    </Alert>
                )}
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
