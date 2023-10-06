import React, { useState, useEffect, ReactElement } from 'react';
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
import { updateOrAddGroup } from 'services/GroupsService';
import { dedupeDelimitedString } from 'utils/textUtils';
import { mergeGroupsWithAuthProviders } from '../AccessControl/AuthProviders/authProviders.utils';
import InviteUsersForm from './InviteUsersForm';
// eslint-disable-next-line import/no-cycle
import InviteUsersConfirmationNoEmail from './InviteUsersConfirmationNoEmail';

type InviteFormValues = {
    emails: string;
    provider: string;
    role: string;
};

export type EmailBuckets = {
    newEmails: string[];
    existingEmails: string[];
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
    const [modalView, setModalView] = useState<'FORM' | 'TEMPLATE' | 'CONFIRM'>('FORM');
    const [emailBuckets, setEmailBuckets] = useState<EmailBuckets | null>(null);
    const [apiError, setApiError] = useState<Error | null>(null);

    const { authProviders, groups, roles, showInviteModal } = useSelector(feedbackState);
    const authProvidersWithRules = mergeGroupsWithAuthProviders(authProviders, groups);

    // TODO: replace this constant with an actual check for an email service
    const isEmailServiceAvailable = false;

    const dispatch = useDispatch();

    const formik = useFormik<InviteFormValues>({
        initialValues: defaultInviteFormValues,
        onSubmit: () => {}, // required but not used, because the submit action is in the modal footer
        validationSchema,
        validateOnMount: true,
    });

    const { isValid, values, setFieldValue, resetForm } = formik;

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
                    lowercasedName.includes('red hat sso') || lowercasedName.includes('openshift')
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
        // check whether any of the listed emails already have rules for this auth provider
        const providerWithRules = authProvidersWithRules.find(
            (provider) => provider.name === values.provider
        );

        if (!providerWithRules) {
            // this should not be possible, but just in case
            throw new Error('selected auth provider for inviting users is no longer available');
        }

        const emailArr = dedupeDelimitedString(values.emails);
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const buckets = emailArr.reduce<EmailBuckets>(
            (acc, email) => {
                if (
                    Array.isArray(providerWithRules.groups) &&
                    providerWithRules.groups.some(
                        (group) => group.props.key === 'email' && group.props.value === email
                    )
                ) {
                    return {
                        newEmails: acc.newEmails,
                        existingEmails: [...acc.existingEmails, email],
                    };
                }
                return { newEmails: [...acc.newEmails, email], existingEmails: acc.existingEmails };
            },
            { newEmails: [], existingEmails: [] }
        );
        setEmailBuckets(buckets);

        if (buckets.newEmails.length === 0) {
            // don't reset form, so that user can come back and try again
            setModalView('TEMPLATE');
        } else {
            // create new auth provider rules
            const requiredGroups = buckets.newEmails.map((newEmail) => ({
                props: {
                    id: '',
                    traits: {
                        mutabilityMode: 'ALLOW_MUTATE',
                        visibility: 'VISIBLE',
                        origin: 'IMPERATIVE',
                    },
                    authProviderId: providerWithRules.id,
                    key: 'email',
                    value: newEmail,
                },
                roleName: values.role,
            }));

            // eslint-disable-next-line no-void
            void updateOrAddGroup({ oldGroups: [], newGroups: requiredGroups })
                .then(() => {
                    // TODO: detect if an email server is available
                    //       1. if so, send emails to the listed recipients
                    //       2. if not, show the emails and the message body for copying to a manual email
                    if (isEmailServiceAvailable) {
                        // TODO send emails
                    }

                    setModalView('TEMPLATE');
                })
                .catch((err) => {
                    setApiError(err);
                });
        }

        // TODO: Show warning if some emails will not be added
    }

    function onClose() {
        resetForm();
        setModalView('FORM');
        setEmailBuckets(null);

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
                            To add an auth provider, visit:
                            <Link onClick={onClose} to={`${accessControlBasePath}/auth-providers`}>
                                Access Control
                            </Link>{' '}
                        </Text>
                    </Alert>
                )}
                {modalView === 'FORM' && (
                    <>
                        {apiError && (
                            <Alert
                                title="Problem inviting the specified users"
                                variant="danger"
                                isInline
                                className="pf-u-mb-lg"
                            >
                                <Text>The following error occurred:</Text>
                                <Text>{apiError?.message}</Text>
                            </Alert>
                        )}
                        <InviteUsersForm
                            formik={formik}
                            providers={authProviders}
                            roles={allowedRoles}
                            onChange={onChange}
                        />
                    </>
                )}
                {modalView === 'TEMPLATE' && emailBuckets !== null && (
                    <InviteUsersConfirmationNoEmail
                        emailBuckets={emailBuckets}
                        onClose={onClose}
                        role={values.role}
                    />
                )}
            </ModalBoxBody>
            <ModalBoxFooter>
                {modalView === 'FORM' && (
                    <>
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
                    </>
                )}
                {modalView === 'TEMPLATE' && (
                    <>
                        {emailBuckets?.newEmails?.length === 0 && (
                            <Button
                                key="done"
                                variant="secondary"
                                onClick={() => setModalView('FORM')}
                                isDisabled={!isValid}
                            >
                                Go back to form
                            </Button>
                        )}
                        <Button
                            key="done"
                            variant="primary"
                            onClick={onClose}
                            isDisabled={!isValid}
                        >
                            Done
                        </Button>
                    </>
                )}
            </ModalBoxFooter>
        </Modal>
    );
}

export default InviteUsersModal;
