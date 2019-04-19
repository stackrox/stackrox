import React from 'react';
import { Link } from 'react-router-dom';
import { stackroxSupport } from 'messages/common';
import { LICENSE_STATUS, LICENSE_UPLOAD_STATUS } from 'reducers/license';
import { licensePath } from 'routePaths';
import { distanceInWordsToNow, differenceInDays } from 'date-fns';

export const noneText = 'In order to use StackRox, please obtain and install a valid license key.';
export const invalidText =
    'Your StackRox license key is invalid. In order to use StackRox, please obtain and install a new valid license key.';
export const expiredText = `Your license key has expired. Please upload a new license key, or contact our customer success team over email or by calling ${
    stackroxSupport.phoneNumber.withSpaces
} to renew  your StackRox Kubernetes Security  Platform license.`;
export const validText = 'Your StackRox license has been renewed';

export const getLicenseStatusMessage = (status, message) => {
    const result = {
        text: '',
        type: 'info'
    };
    if (!status && !message) return null;
    switch (status) {
        case LICENSE_UPLOAD_STATUS.VERIFYING:
            result.text = 'Verifying...';
            result.type = 'info';
            break;
        case LICENSE_STATUS.VALID:
            result.text = message || validText;
            result.type = 'info';
            break;
        case LICENSE_STATUS.RESTARTING:
            result.text = 'Restarting...';
            result.type = 'info';
            break;
        case LICENSE_STATUS.NONE:
            result.text = noneText;
            result.type = 'warn';
            break;
        case LICENSE_STATUS.EXPIRED:
            result.text = expiredText;
            result.type = 'warn';
            break;
        default:
            result.text = message || invalidText;
            result.type = 'error';
            break;
    }
    return result;
};

const getExpirationMessageType = expirationDate => {
    const daysLeft = differenceInDays(expirationDate, new Date());
    if (daysLeft > 3 && daysLeft <= 14) {
        return 'warn';
    }
    if (daysLeft <= 3) {
        return 'error';
    }
    return null;
};

const createExpirationMessage = (expirationDate, message) => {
    const type = getExpirationMessageType(expirationDate);
    return {
        message,
        type
    };
};

export const createExpirationMessageWithLink = expirationDate => {
    const type = getExpirationMessageType(expirationDate);
    const message = (
        <div>
            Your license will expire in {distanceInWordsToNow(expirationDate)}.
            <Link
                className={`mx-1 ${type === 'warn' ? 'text-warning-800' : 'text-alert-800'}`}
                to={licensePath}
            >
                Upload a new license key
            </Link>
            to renew your account.
        </div>
    );
    return createExpirationMessage(expirationDate, message);
};

export const createExpirationMessageWithoutLink = expirationDate => {
    const message = `Your license will expire in ${distanceInWordsToNow(
        expirationDate
    )}. Upload a new license key to renew your account.`;
    return createExpirationMessage(expirationDate, message);
};
