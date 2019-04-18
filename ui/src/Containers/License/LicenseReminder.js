import React, { useState, useEffect } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { withRouter } from 'react-router-dom';
import { createExpirationMessage } from 'Containers/License/helpers';
import {
    differenceInDays,
    differenceInHours,
    differenceInMinutes,
    differenceInSeconds
} from 'date-fns';
import { licenseStartUpPath } from 'routePaths';

import MessageBanner from 'Components/MessageBanner';

const getDelay = expirationDate => {
    const dateNow = new Date();

    const daysLeft = differenceInDays(expirationDate, dateNow);
    if (daysLeft >= 2) return 1000 * 60 * 60 * 24;

    const hoursLeft = differenceInHours(expirationDate, dateNow);
    if (hoursLeft >= 2) return 1000 * 60 * 60;

    const minutesLeft = differenceInMinutes(expirationDate, dateNow);
    if (minutesLeft >= 2) return 1000 * 60;

    const secondsLeft = differenceInSeconds(expirationDate, dateNow);
    if (secondsLeft >= 1) return 1000;

    return null;
};

const LicenseReminder = ({ expirationDate, history }) => {
    const [showReminder, setReminder] = useState(true);
    const [expirationMessage, setExpirationMessage] = useState(
        createExpirationMessage(expirationDate)
    );

    if (!showReminder) return null;
    if (!expirationMessage) return null;

    const onCancelHandler = () => () => setReminder(false);

    useEffect(() => {
        const delay = getDelay(expirationDate);
        let timerID;
        if (delay) {
            timerID = setInterval(
                () => setExpirationMessage(createExpirationMessage(expirationDate)),
                delay
            );
        } else {
            history.push(licenseStartUpPath);
        }
        return function cleanup() {
            clearInterval(timerID);
        };
    });

    const { type, message } = expirationMessage;
    return (
        <MessageBanner
            type={type}
            message={message}
            showCancel={type === 'warn'}
            onCancel={onCancelHandler()}
        />
    );
};

LicenseReminder.propTypes = {
    expirationDate: PropTypes.string,
    history: ReactRouterPropTypes.history
};

LicenseReminder.defaultProps = {
    expirationDate: null,
    history: null
};

const mapStateToProps = createStructuredSelector({
    expirationDate: selectors.getLicenseExpirationDate
});

export default connect(
    mapStateToProps,
    null
)(withRouter(LicenseReminder));
