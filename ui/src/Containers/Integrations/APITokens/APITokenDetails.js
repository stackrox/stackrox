import React from 'react';
import * as Icon from 'react-feather';
import { CopyToClipboard } from 'react-copy-to-clipboard';

import PropTypes from 'prop-types';
import dateFns from 'date-fns';
import LabeledValue from 'Components/LabeledValue';
import dateTimeFormat from 'constants/dateTimeFormat';

const formatDate = date => dateFns.format(date, dateTimeFormat);

const Token = ({ token }) => {
    if (!token) return null;
    return (
        <div className="flex flex-col items-end">
            <div className="flex">
                <span className="flex-grow">
                    Please copy the generated token and store it safely. You will not be able to
                    access it again after you close this window.
                </span>
                <CopyToClipboard text={token}>
                    <button type="button" className="btn-success h-8 w-8">
                        {<Icon.Copy className="h-4 w-4" />}
                    </button>
                </CopyToClipboard>
            </div>
            <span className="bg-tertiary-200 word-break-all">{token}</span>
        </div>
    );
};

Token.propTypes = {
    token: PropTypes.string
};

Token.defaultProps = {
    token: ''
};

const APITokenDetails = ({ token, metadata }) => (
    <div className="p-4 w-full" data-test-id="api-token-details">
        <Token token={token} />
        <LabeledValue label="Name" value={metadata.name} />
        <LabeledValue label="Role" value={metadata.role} />
        <LabeledValue label="Issued" value={formatDate(metadata.issuedAt)} />
        <LabeledValue label="Expiration" value={formatDate(metadata.expiration)} />
        <LabeledValue label="Revoked" value={metadata.revoked ? 'Yes' : 'No'} />
    </div>
);

APITokenDetails.propTypes = {
    token: PropTypes.string,
    metadata: PropTypes.shape({
        name: PropTypes.string.isRequired,
        role: PropTypes.string.isRequired,
        issuedAt: PropTypes.string.isRequired,
        expiration: PropTypes.string.isRequired,
        revoked: PropTypes.bool.isRequired
    }).isRequired
};

APITokenDetails.defaultProps = {
    token: ''
};

export default APITokenDetails;
