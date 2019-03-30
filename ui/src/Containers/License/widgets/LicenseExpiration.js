import React from 'react';
import PropTypes from 'prop-types';
import { format, distanceInWordsToNow, differenceInDays } from 'date-fns';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import * as Icon from 'react-feather';
import Widget from 'Components/Widget';
import Message from 'Components/Message';
import UploadLicense from 'Containers/License/UploadLicense';

const getExpirationMessage = expirationDate => {
    const message = `Your license will expire in ${distanceInWordsToNow(
        expirationDate
    )}. Upload a new license key to renew your account.`;
    const type = differenceInDays(expirationDate, new Date()) < 3 ? 'error' : 'warn';
    return {
        message,
        type
    };
};

const LicenseExpiration = ({ expirationDate }) => {
    const expirationMessage = getExpirationMessage(expirationDate);
    return (
        <Widget header="License Expiration">
            <div className="py-4 px-6 w-full">
                <div className="flex items-center text-lg pb-4 border-b border-base-300">
                    <Icon.Clock className="h-5 w-5 text-primary-800 text-4xl mr-4" />
                    <div className="text-primary-800 font-400 text-4xl">
                        {format(expirationDate, 'MM/DD/YY')}
                    </div>
                    <div className="flex flex-1 justify-end text-base-500">
                        ({distanceInWordsToNow(expirationDate)} from now)
                    </div>
                </div>
                <div className="text-center">
                    <Message type={expirationMessage.type} message={expirationMessage.message} />
                    <UploadLicense />
                </div>
            </div>
        </Widget>
    );
};

LicenseExpiration.propTypes = {
    expirationDate: PropTypes.string
};

LicenseExpiration.defaultProps = {
    expirationDate: null
};

const mapStateToProps = createStructuredSelector({
    expirationDate: selectors.getLicenseExpirationDate
});

export default connect(
    mapStateToProps,
    null
)(LicenseExpiration);
