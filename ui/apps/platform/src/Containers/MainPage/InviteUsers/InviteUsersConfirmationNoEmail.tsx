import type { ReactElement } from 'react';
import { Alert, ClipboardCopy, Content } from '@patternfly/react-core';
import { Link } from 'react-router-dom-v5-compat';

import { accessControlBasePath } from 'routePaths';
import type { BucketsForNewAndExistingEmails } from './InviteUsers.utils';

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
                    component="p"
                    variant="warning"
                    isInline
                    className="pf-v6-u-mb-lg"
                >
                    <Content component="p">
                        You must enter at least one email that does not yet have a rule in the
                        system.
                    </Content>
                    <Content component="p">
                        Visit the{' '}
                        <Link onClick={onClose} to={`${accessControlBasePath}/auth-providers`}>
                            auth provider
                        </Link>{' '}
                        section to check which users already have rules.
                    </Content>
                </Alert>
            ) : (
                <>
                    {emailBuckets.existingEmails.length > 0 && (
                        <Alert
                            title="Some emails already have auth provider rules."
                            component="p"
                            variant="warning"
                            isInline
                            className="pf-v6-u-mb-lg"
                        >
                            <Content component="p" className="pf-v6-u-mb-md">
                                The following users could not be invited because their emails
                                already have rules applied to them.
                            </Content>
                            <Content component="p" className="pf-v6-u-mb-md">
                                {emailBuckets.existingEmails.join(', ')}
                            </Content>
                            <Content component="p">
                                Visit the{' '}
                                <Link
                                    onClick={onClose}
                                    to={`${accessControlBasePath}/auth-providers`}
                                >
                                    auth provider
                                </Link>{' '}
                                section to check these rules.
                            </Content>
                        </Alert>
                    )}
                    <Content component="p" className="pf-v6-u-mb-sm">
                        New rules have been created, but invitation emails could not be sent. Use
                        the text below to manually send emails to your invitees.
                    </Content>
                    <Content component="p" className="pf-v6-u-mb-lg">
                        Role: <strong>{role}</strong>
                    </Content>
                    <ClipboardCopy
                        isReadOnly
                        isExpanded
                        hoverTip="Copy"
                        clickTip="Copied"
                        variant="expansion"
                        className="pf-v6-u-mb-md"
                    >
                        {emailBuckets.newEmails.join(', ')}
                    </ClipboardCopy>
                    <ClipboardCopy
                        isReadOnly
                        isExpanded
                        hoverTip="Copy"
                        clickTip="Copied"
                        variant="expansion"
                        className="pf-v6-u-mb-md"
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
