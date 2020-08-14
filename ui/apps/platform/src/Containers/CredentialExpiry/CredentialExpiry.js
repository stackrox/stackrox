import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { differenceInDays, distanceInWordsStrict, format } from 'date-fns';

import MessageBanner from 'Components/MessageBanner';
import { selectors } from 'reducers';
import Button from '../../Components/Button';

const getExpirationMessageType = (daysLeft) => {
    if (daysLeft > 14) {
        return 'info';
    }
    if (daysLeft > 3) {
        return 'warn';
    }
    return 'error';
};

const CredentialExpiry = ({
    component,
    expiryFetchFunc,
    shouldHaveReadWritePermission,
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
    const hasServiceIdentityWritePermission = shouldHaveReadWritePermission('ServiceIdentity');
    const message = (
        <span className="flex-1 text-center">
            The {component} certificate expires in {distanceInWordsStrict(expirationDate, now)} on{' '}
            {format(expirationDate, 'MMMM D, YYYY')} (at {format(expirationDate, 'h:mm a')}).{' '}
            {hasServiceIdentityWritePermission ? (
                <>
                    To use renewed certificates,{' '}
                    <Button
                        text="download this YAML file"
                        className="text-tertiary-700 hover:text-tertiary-800 underline font-700 justify-center"
                        onClick={downloadYAMLFunc}
                    />{' '}
                    and apply it to your cluster.
                </>
            ) : (
                'Contact your administrator.'
            )}
        </span>
    );

    return (
        <MessageBanner
            dataTestId={`cert-expiry-banner-${component.split(' ').join('-').toLowerCase()}`}
            type={type}
            component={message}
            showCancel={type === 'warn'}
        />
    );
};

CredentialExpiry.propTypes = {
    component: PropTypes.string.isRequired,
    expiryFetchFunc: PropTypes.func.isRequired,
    shouldHaveReadWritePermission: PropTypes.func.isRequired,
    downloadYAMLFunc: PropTypes.func.isRequired,
};

const mapStateToProps = createStructuredSelector({
    shouldHaveReadWritePermission: selectors.shouldHaveReadWritePermission,
});

export default connect(mapStateToProps, null)(CredentialExpiry);
