import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as licenseActions } from 'reducers/license';
import { distanceInWordsToNow, differenceInDays } from 'date-fns';

import MessageBanner from 'Components/MessageBanner';

const getExpirationMessage = expirationDate => {
    const daysLeft = differenceInDays(expirationDate, new Date());
    const message = `Your license will expire in ${distanceInWordsToNow(
        expirationDate
    )}. Upload a new license key to renew your account.`;
    let type;

    if (daysLeft > 3 && daysLeft <= 14) {
        type = 'warn';
    } else if (daysLeft <= 3) {
        type = 'error';
    } else {
        return null;
    }
    return {
        message,
        type
    };
};

const LicenseReminder = ({ showLicenseReminder, expirationDate, dismissLicenseReminder }) => {
    if (!showLicenseReminder) return null;
    const expirationMessage = getExpirationMessage(expirationDate);
    if (!expirationMessage) return null;
    const { type, message } = expirationMessage;
    return (
        <MessageBanner
            type={type}
            message={message}
            showCancel={type === 'warn'}
            onCancel={dismissLicenseReminder}
        />
    );
};

LicenseReminder.propTypes = {
    expirationDate: PropTypes.string,
    showLicenseReminder: PropTypes.bool.isRequired,
    dismissLicenseReminder: PropTypes.func.isRequired
};

LicenseReminder.defaultProps = {
    expirationDate: null
};

const mapStateToProps = createStructuredSelector({
    expirationDate: selectors.getLicenseExpirationDate,
    showLicenseReminder: selectors.shouldShowLicenseReminder
});

const mapDispatchToProps = {
    dismissLicenseReminder: licenseActions.dismissLicenseReminder
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(LicenseReminder);
