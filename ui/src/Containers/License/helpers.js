import { stackroxSupport } from 'messages/common';
import { LICENSE_STATUS, LICENSE_UPLOAD_STATUS } from 'reducers/license';

export const invalidText =
    'Your StackRox license key is invalid. In order to use StackRox, please obtain and install a new valid license key.';
export const expiredText = `Your license key has expired. Please upload a new license key, or contact our customer success team over email or by calling ${
    stackroxSupport.phoneNumber.withSpaces
} to renew  your StackRox Kubernetes Security  Platform license.`;
export const validText = 'Your StackRox license has was renewed';

export const getUploadResponseMessage = status => {
    const message = {
        text: '',
        type: 'info'
    };
    switch (status) {
        case LICENSE_UPLOAD_STATUS.VALID:
            message.text = validText;
            message.type = 'info';
            return message;
        case LICENSE_UPLOAD_STATUS.EXPIRED:
            message.text = expiredText;
            message.type = 'error';
            return message;
        case LICENSE_UPLOAD_STATUS.INVALID:
            message.text = invalidText;
            message.type = 'error';
            return message;
        default:
            return message;
    }
};

export const getLicenseStatusMessage = licenseStatus => {
    let message = {
        text: '',
        type: 'info'
    };
    switch (licenseStatus) {
        case LICENSE_STATUS.EXPIRED:
        case LICENSE_STATUS.REVOKED:
            message.text = expiredText;
            message.type = 'error';
            return message;
        case LICENSE_STATUS.UNKNOWN:
        case LICENSE_STATUS.OTHER:
        case LICENSE_STATUS.NOT_YET_VALID:
            message.text = invalidText;
            message.type = 'error';
            return message;
        default:
            message = null;
            return message;
    }
};
