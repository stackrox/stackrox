import React, { useState, useEffect } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { getHasReadPermission } from 'reducers/roles';
import { withRouter } from 'react-router-dom';
import {
    createExpirationMessageWithLink,
    createExpirationMessageWithoutLink,
} from 'Containers/License/helpers';

import MessageBanner from 'Components/MessageBanner';

const LicenseReminder = ({ expirationDate, userRolePermissions }) => {
    const createExpirationMessage = getHasReadPermission('Licenses', userRolePermissions)
        ? createExpirationMessageWithLink
        : createExpirationMessageWithoutLink;

    const [showReminder, setReminder] = useState(true);
    const [expirationMessage, setExpirationMessage] = useState(null);

    useEffect(() => {
        if (!expirationDate) {
            setExpirationMessage(null);
            return;
        }
        setExpirationMessage(createExpirationMessage(expirationDate));
    }, [createExpirationMessage, expirationDate]);

    if (!getHasReadPermission('Licenses', userRolePermissions)) {
        return null;
    }
    if (!showReminder) {
        return null;
    }
    if (!expirationMessage || expirationMessage.type === 'base') {
        return null;
    }

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
    userRolePermissions: PropTypes.shape({ globalAccess: PropTypes.string.isRequired }),
};

LicenseReminder.defaultProps = {
    expirationDate: null,
    userRolePermissions: null,
};

const mapStateToProps = createStructuredSelector({
    expirationDate: selectors.getLicenseExpirationDate,
    userRolePermissions: selectors.getUserRolePermissions,
});

export default connect(mapStateToProps, null)(withRouter(LicenseReminder));
