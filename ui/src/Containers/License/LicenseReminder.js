import React, { useState, useEffect } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { withRouter } from 'react-router-dom';
import {
    createExpirationMessageWithLink,
    createExpirationMessageWithoutLink
} from 'Containers/License/helpers';

import MessageBanner from 'Components/MessageBanner';

const LicenseReminder = ({ expirationDate, shouldHaveReadPermission }) => {
    const createExpirationMessage = shouldHaveReadPermission('Licenses')
        ? createExpirationMessageWithLink
        : createExpirationMessageWithoutLink;

    const [showReminder, setReminder] = useState(true);
    const [expirationMessage, setExpirationMessage] = useState(null);

    useEffect(
        () => {
            if (!expirationDate) {
                setExpirationMessage(null);
                return;
            }
            setExpirationMessage(createExpirationMessage(expirationDate));
        },
        [createExpirationMessage, expirationDate]
    );

    if (!shouldHaveReadPermission('Licenses')) return null;
    if (!showReminder) return null;
    if (!expirationMessage || expirationMessage.type === 'info') return null;

    const onCancelHandler = () => () => setReminder(false);

    const { type, message } = expirationMessage;
    return (
        <MessageBanner
            type={type}
            component={message}
            showCancel={type === 'warn'}
            onCancel={onCancelHandler()}
        />
    );
};

LicenseReminder.propTypes = {
    expirationDate: PropTypes.string,
    shouldHaveReadPermission: PropTypes.func.isRequired
};

LicenseReminder.defaultProps = {
    expirationDate: null
};

const mapStateToProps = createStructuredSelector({
    expirationDate: selectors.getLicenseExpirationDate,
    shouldHaveReadPermission: selectors.shouldHaveReadPermission
});

export default connect(
    mapStateToProps,
    null
)(withRouter(LicenseReminder));
