import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { differenceInDays, distanceInWordsStrict, format } from 'date-fns';
import { Banner, Button } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { getHasReadWritePermission } from 'reducers/roles';

const getExpirationMessageType = (daysLeft) => {
    if (daysLeft > 14) {
        return 'info';
    }
    if (daysLeft > 3) {
        return 'warning';
    }
    return 'danger';
};

const CredentialExpiry = ({
    component,
    expiryFetchFunc,
    userRolePermissions,
    downloadYAMLFunc,
}) => {
    const [expirationDate, setExpirationDate] = useState(null);
    useEffect(() => {
        expiryFetchFunc()
            .then((expiry) => {
                setExpirationDate(expiry);
            })
            .catch((e) => {
                // ignored because it's either a temporary network issue,
                //   or symptom of a larger problem
                // Either way, we don't want to spam the logimbue service

                // eslint-disable-next-line no-console
                console.warn(`Problem checking the certification expiration for ${component}.`, e);
            });
    }, [expiryFetchFunc, component]);

    if (!expirationDate) {
        return null;
    }
    const now = new Date();
    const type = getExpirationMessageType(differenceInDays(expirationDate, now));
    if (type === 'info') {
        return null;
    }
    const hasServiceIdentityWritePermission = getHasReadWritePermission(
        'ServiceIdentity',
        userRolePermissions
    );
    const downloadLink = (
        <Button variant="link" isInline onClick={downloadYAMLFunc}>
            download this YAML file
        </Button>
    );
    const message = (
        <span className="flex-1 text-center">
            {component} certificate expires in {distanceInWordsStrict(expirationDate, now)} on{' '}
            {format(expirationDate, 'MMMM D, YYYY')} (at {format(expirationDate, 'h:mm a')}).{' '}
            {hasServiceIdentityWritePermission ? (
                <>To use renewed certificates, {downloadLink} and apply it to your cluster.</>
            ) : (
                'Contact your administrator.'
            )}
        </span>
    );

    return (
        <Banner className="pf-u-text-align-center" isSticky variant={type}>
            {message}
        </Banner>
    );
};

CredentialExpiry.propTypes = {
    component: PropTypes.string.isRequired,
    expiryFetchFunc: PropTypes.func.isRequired,
    userRolePermissions: PropTypes.shape({
        resourceToAccess: PropTypes.shape({}),
    }),
    downloadYAMLFunc: PropTypes.func.isRequired,
};

CredentialExpiry.defaultProps = {
    userRolePermissions: null,
};

const mapStateToProps = createStructuredSelector({
    userRolePermissions: selectors.getUserRolePermissions,
});

export default connect(mapStateToProps, null)(CredentialExpiry);
