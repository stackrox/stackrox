import React, { ReactElement } from 'react';
import { Alert, ClipboardCopy, ClipboardCopyVariant, Text } from '@patternfly/react-core';
import { Link } from 'react-router-dom';

import { accessControlBasePath } from 'routePaths';
import { BucketsForNewAndExistingEmails } from './InviteUsers.utils';

type InviteUsersConfirmationNoEmailProps = {
    emailBuckets: BucketsForNewAndExistingEmails;
    onClose: () => void;
    role: string;
};

function InviteUsersConfirmationNoEmail({
    emailBuckets,
    onClose,
    role,
}: InviteUsersConfirmationNoEmailProps): ReactElement {
    return (
        <div>
            {emailBuckets.newEmails.length === 0 ? (
                <Alert
                    title="All entered emails already have auth provider rules"
                    variant="warning"
                    isInline
                    className="pf-u-mb-lg"
                >
                    <Text>
                        You must enter at least one email that does not yet have a rule in the
                        system.
                    </Text>
                    <Text>
                        Visit the{' '}
                        <Link onClick={onClose} to={`${accessControlBasePath}/auth-providers`}>
                            auth provider
                        </Link>{' '}
                        section to check which users already have rules.
                    </Text>
                </Alert>
            ) : (
                <>
                    {emailBuckets.existingEmails.length > 0 && (
                        <Alert
                            title="Some emails already have auth provider rules."
                            variant="warning"
                            isInline
                            className="pf-u-mb-lg"
                        >
                            <Text className="pf-u-mb-md">
                                The following users could not be invited because their emails
                                already have rules applied to them.
                            </Text>
                            <Text className="pf-u-mb-md">
                                {emailBuckets.existingEmails.join(', ')}
                            </Text>
                            <Text>
                                Visit the{' '}
                                <Link
                                    onClick={onClose}
                                    to={`${accessControlBasePath}/auth-providers`}
                                >
                                    auth provider
                                </Link>{' '}
                                section to check these rules.
                            </Text>
                        </Alert>
                    )}
                    <Text className="pf-u-mb-sm">
                        New rules have been created, but invitation emails could not be sent. Use
                        the text below to manually send emails to your invitees.
                    </Text>
                    <Text className="pf-u-mb-lg">
                        Role: <strong>{role}</strong>
                    </Text>
                    <ClipboardCopy
                        isReadOnly
                        isExpanded
                        hoverTip="Copy"
                        clickTip="Copied"
                        variant={ClipboardCopyVariant.expansion}
                        className="pf-u-mb-md"
                    >
                        {emailBuckets.newEmails.join(', ')}
                    </ClipboardCopy>
                    <ClipboardCopy
                        isReadOnly
                        isExpanded
                        hoverTip="Copy"
                        clickTip="Copied"
                        variant={ClipboardCopyVariant.expansion}
                        className="pf-u-mb-md"
                    >
                        You have been invited to use Red Hat Advanced Cluster Security. Please use
                        the link to sign in: {window.location.origin}
                    </ClipboardCopy>
                </>
            )}
        </div>
    );
}

export default InviteUsersConfirmationNoEmail;
