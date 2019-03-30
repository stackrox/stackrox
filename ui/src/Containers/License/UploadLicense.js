import React, { useState, useEffect } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions, LICENSE_UPLOAD_STATUS } from 'reducers/license';
import { getUploadResponseMessage } from 'Containers/License/helpers';

import UploadButton from 'Components/UploadButton';
import Dialog from 'Components/Dialog';

const UploadLicense = ({ licenseUploadStatus, activateLicense }) => {
    const defaultDialogState = !!licenseUploadStatus;
    const defaultVerifyingLicenseState = licenseUploadStatus
        ? licenseUploadStatus === LICENSE_UPLOAD_STATUS.VERIFYINGs
        : false;

    const [dialogMessage, setDialogMessage] = useState(
        getUploadResponseMessage(licenseUploadStatus)
    );
    const [isDialogOpen, openDialog] = useState(defaultDialogState);
    const [isVerifyingLicense, verifyLicense] = useState(defaultVerifyingLicenseState);

    useEffect(
        () => {
            if (licenseUploadStatus !== LICENSE_UPLOAD_STATUS.VERIFYING) {
                verifyLicense(false);
                setDialogMessage(getUploadResponseMessage(licenseUploadStatus));
            }
        },
        [licenseUploadStatus]
    );

    function onUploadHandler(data) {
        verifyLicense(true);
        openDialog(true);
        activateLicense(data);
    }

    function onDialogCancel() {
        openDialog(false);
    }

    return (
        <>
            <UploadButton
                className="p-3 px-6 rounded-sm bg-primary-600 hover:bg-primary-700 text-base-100 uppercase text-center tracking-wide mt-4"
                text="Upload New License Key"
                onChange={onUploadHandler}
            />
            <Dialog
                isOpen={isDialogOpen}
                text={dialogMessage.text}
                cancelText="Ok"
                onCancel={onDialogCancel}
                isLoading={isVerifyingLicense}
                loadingText="Verifying License Key"
            />
        </>
    );
};

UploadLicense.propTypes = {
    licenseUploadStatus: PropTypes.string,
    activateLicense: PropTypes.func.isRequired
};

UploadLicense.defaultProps = {
    licenseUploadStatus: null
};

const mapStateToProps = createStructuredSelector({
    licenseUploadStatus: selectors.getLicenseUploadStatus
});

const mapDispatchToProps = {
    activateLicense: actions.activateLicense
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(UploadLicense);
