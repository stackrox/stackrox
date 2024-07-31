import React, { ReactElement } from 'react';
import {
    Alert,
    Button,
    DescriptionList,
    DescriptionListTerm,
    DescriptionListGroup,
    DescriptionListDescription,
} from '@patternfly/react-core';
import CopyIcon from '@patternfly/react-icons/dist/js/icons/copy-icon';
import { CopyToClipboard } from 'react-copy-to-clipboard';

import { getDateTime } from 'utils/dateUtils';

export type ApiTokenFormResponseMessage = {
    message: string;
    isError: boolean;
    responseData?: {
        response: {
            metadata: {
                expiration: string;
                id: string;
                issuedAt: string;
                name: string;
                revoked: string;
                role: string;
                roles: string[];
            };
            token: string;
        };
    };
};

export type ApiTokenFormMessageAlertProps = {
    message: ApiTokenFormResponseMessage;
};

function ApiTokenResponseDetails({ message }) {
    const { metadata, token } = message.responseData.response;
    return (
        <DescriptionList>
            <DescriptionListGroup>
                <DescriptionListTerm>
                    Please copy the generated token and store it safely. You will not be able to
                    access it again after you close this window.
                    <CopyToClipboard text={token} className="pf-v5-u-ml-sm">
                        <Button variant="control" aria-label="Copy">
                            <CopyIcon />
                        </Button>
                    </CopyToClipboard>
                </DescriptionListTerm>
                <DescriptionListDescription>{token}</DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Expiration</DescriptionListTerm>
                <DescriptionListDescription>
                    {getDateTime(metadata.expiration)}
                </DescriptionListDescription>
            </DescriptionListGroup>
        </DescriptionList>
    );
}

function ApiTokenFormMessageAlert({ message }: ApiTokenFormMessageAlertProps): ReactElement {
    return (
        <Alert
            isInline
            variant={message.isError ? 'danger' : 'success'}
            title={message.message}
            component="p"
        >
            {message.responseData && <ApiTokenResponseDetails message={message} />}
        </Alert>
    );
}

export default ApiTokenFormMessageAlert;
