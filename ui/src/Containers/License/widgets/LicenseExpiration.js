import React, { useState } from 'react';
import { format, distanceInWordsToNow, differenceInDays } from 'date-fns';
import { expirationDate } from 'mockData/licenseData';
import { stackroxSupport } from 'messages/common';

import * as Icon from 'react-feather';
import Widget from 'Components/Widget';
import Message from 'Components/Message';
import UploadButton from 'Components/UploadButton';
import Dialog from 'Components/Dialog';

const getExpirationMessage = () => {
    const message = `Your license will expire in ${distanceInWordsToNow(
        expirationDate
    )}. Upload a new license key to renew your account.`;
    const type = differenceInDays(expirationDate, new Date()) < 3 ? 'error' : 'warn';
    return {
        message,
        type
    };
};

const LicenseExpiration = () => {
    const [isDialogOpen, openDialog] = useState(false);
    const [isVerifyingLicense, verifyLicense] = useState(false);
    const [dialogText, setDialogText] = useState('');

    const expirationMessage = getExpirationMessage();

    const onDownloadHandler = () => () => {
        verifyLicense(true);
        openDialog(true);
        setTimeout(() => {
            const state = Math.random() * 1;
            if (state >= 0.6) {
                setDialogText(
                    'Your StackRox license key is invalid. In order to use StackRox, please obtain and install a new valid license key.'
                );
            } else if (state >= 0.3 && state < 0.6) {
                setDialogText(
                    `Your StackRox license has expired. Please contact our customer success team at ${
                        stackroxSupport.phoneNumber.withSpaces
                    } to continue using the product.`
                );
            } else {
                setDialogText('Your StackRox license has been successfully renewed.');
            }
            verifyLicense(false);
        }, 2000);
    };
    const onDialogCancel = () => () => {
        openDialog(false);
    };

    return (
        <Widget header="License Expiration">
            <div className="py-4 px-6 w-full">
                <div className="flex items-center text-lg pb-4 border-b border-base-300">
                    <Icon.Clock className="h-5 w-5 text-primary-800 text-4xl mr-4" />
                    <div className="text-primary-800 font-400 text-4xl">
                        {format(expirationDate, 'MM/DD/YY')}
                    </div>
                    <div className="flex flex-1 justify-end text-base-500">
                        ({distanceInWordsToNow(expirationDate)})
                    </div>
                </div>
                <div className="text-center">
                    <Message type={expirationMessage.type} message={expirationMessage.message} />
                    <UploadButton
                        className="p-3 px-6 rounded-sm bg-primary-600 hover:bg-primary-700 text-base-100 uppercase text-center tracking-wide mt-4"
                        text="Upload New License Key"
                        onChange={onDownloadHandler()}
                    />
                </div>
            </div>
            <Dialog
                isOpen={isDialogOpen}
                text={dialogText}
                onCancel={onDialogCancel()}
                cancelText="Ok"
                isLoading={isVerifyingLicense}
                loadingText="Verifying License Key"
            />
        </Widget>
    );
};

export default LicenseExpiration;
