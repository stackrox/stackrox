import React from 'react';
import PropTypes from 'prop-types';
import { Redirect, Link } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { LICENSE_STATUS } from 'reducers/license';

import { getLicenseStatusMessage } from 'Containers/License/helpers';
import { dashboardPath } from 'routePaths';
import logoPlatform from 'images/logo-platform.svg';

import MessageBanner from 'Components/MessageBanner';
import LoadingSection from 'Components/LoadingSection';
import UploadLicense from 'Containers/License/UploadLicense';

const getDefaultBannerText = (licenseUploadStatus, licenseStatus) => {
    // if we have an upload status, show that
    if (licenseUploadStatus) {
        const { status, message } = licenseUploadStatus;
        return getLicenseStatusMessage(status, message);
    }
    // if we haven't uploaded, show what the license status is currently
    return getLicenseStatusMessage(licenseStatus, null);
};

const StartUpScreen = ({ licenseStatus, licenseUploadStatus }) => {
    const fetchingLicense = licenseStatus === LICENSE_STATUS.RESTARTING;

    if (licenseStatus === LICENSE_STATUS.VALID) {
        return <Redirect to="/" />;
    }

    if (fetchingLicense) {
        return <LoadingSection message="Verifying License..." />;
    }

    const hasLicense = licenseStatus === LICENSE_STATUS.VALID;

    const bannerMessage = getDefaultBannerText(licenseUploadStatus, licenseStatus);

    const message = (
        <div className="flex flex-col items-center bg-base-100 w-2/5 md:w-3/5 xl:w-2/5 relative overflow-hidden rounded-t">
            {bannerMessage && (
                <MessageBanner type={bannerMessage.type} message={bannerMessage.text} />
            )}
        </div>
    );

    const button = hasLicense ? (
        <Link
            className="p-3 px-6 rounded-sm bg-primary-600 hover:bg-primary-700 text-base-100 uppercase text-center tracking-wide mt-4 no-underline"
            to={dashboardPath}
        >
            Go to Dashboard
        </Link>
    ) : (
        <UploadLicense licenseUploadStatus={{ status: licenseStatus }} isStartUpScreen />
    );

    return (
        <section className="flex flex-col items-center justify-center h-full bg-primary-800">
            {message}
            <div className="flex flex-col items-center justify-center bg-base-100 w-2/5 md:w-3/5 xl:w-2/5 relative login-bg rounded">
                <div className="login-border-t h-1 w-full" />
                <div className="flex flex-col items-center justify-center w-full">
                    <img className="h-40 h-40 py-6" src={logoPlatform} alt="StackRox" />
                </div>
                <div className="border-t border-base-300 p-6 w-full text-center">{button}</div>
            </div>
        </section>
    );
};

StartUpScreen.propTypes = {
    licenseStatus: PropTypes.string,
    licenseUploadStatus: PropTypes.shape({
        status: PropTypes.string,
        message: PropTypes.string
    })
};

StartUpScreen.defaultProps = {
    licenseStatus: null,
    licenseUploadStatus: PropTypes.shape({
        status: null,
        message: ''
    })
};

const mapStateToProps = createStructuredSelector({
    licenseStatus: selectors.getLicenseStatus,
    licenseUploadStatus: selectors.getLicenseUploadStatus
});

export default connect(
    mapStateToProps,
    null
)(StartUpScreen);
