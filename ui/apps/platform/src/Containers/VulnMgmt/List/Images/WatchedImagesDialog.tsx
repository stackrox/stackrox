import React, { ReactElement, useState, useEffect } from 'react';
import { Formik, ErrorMessage } from 'formik';
import * as Yup from 'yup';
import { ClipLoader } from 'react-spinners';
import { XCircle } from 'react-feather';
import { Button, CondensedAlertButton, Message } from '@stackrox/ui-components';

import CustomDialogue from 'Components/CustomDialogue';
import {
    labelClassName,
    sublabelClassName,
    wrapperMarginClassName,
    inputTextClassName,
} from 'constants/form.constants';
import { getWatchedImages, watchImage, unwatchImage } from 'services/imageService';
import { WatchedImage } from 'types/image.proto';

type WatchedImagesDialogProps = {
    closeDialog: () => void;
};

const WatchedImagesDialog = ({ closeDialog }: WatchedImagesDialogProps): ReactElement => {
    const [currentWatchedImages, setCurrentWatchedImages] = useState<WatchedImage[]>([]);
    const [successMessage, setSuccessMessage] = useState<ReactElement | string>('');
    const [errorMessage, setErrorMessage] = useState<ReactElement | string>('');

    function refreshWatchList() {
        getWatchedImages()
            .then((images) => {
                setCurrentWatchedImages(images);
            })
            .catch((error) => {
                setErrorMessage(
                    error?.message ||
                        'An error occurred retrieving the current list of watched images'
                );
            });
    }

    useEffect(() => {
        refreshWatchList();
    }, []);

    function addToWatch(values, { setSubmitting }) {
        setSuccessMessage('');
        setErrorMessage('');
        watchImage(values.imageTag)
            .then((normalizedName) => {
                setSuccessMessage(
                    <div>
                        <strong>{normalizedName}</strong>
                        {normalizedName !== values.imageTag && (
                            <>
                                {` (normalized form of `}
                                <strong>{values.imageTag}</strong>)
                            </>
                        )}{' '}
                        has been added to the list of images to be scanned.
                    </div>
                );
                refreshWatchList();
            })
            .catch((error) => {
                setErrorMessage(
                    <div>
                        <p className="mb-2">
                            The image name you submitted, <strong>{values.imageTag}</strong> could
                            not be processed.
                        </p>
                        <p className="mb-1">The server response with the following message.</p>
                        <blockquote className="p-4 border-l-4 quote">{error.message}</blockquote>
                    </div>
                );
            })
            .finally(() => {
                setSubmitting(false);
            });
    }

    function getRemoveFromWatch(imageName) {
        return function removeFromWatch() {
            setSuccessMessage('');
            setErrorMessage('');
            unwatchImage(imageName)
                .then(() => {
                    setSuccessMessage(
                        <p>
                            <strong>{imageName}</strong> is no longer being watched.
                        </p>
                    );
                    refreshWatchList();
                })
                .catch((error) => {
                    setErrorMessage(
                        <div>
                            <p className="mb-1">Could not remove the image from the watch list.</p>
                            <blockquote className="p-4 border-l-4 quote">
                                {error?.message || 'Unknown error'}
                            </blockquote>
                        </div>
                    );
                });
        };
    }

    const imageList = currentWatchedImages
        .sort((a: WatchedImage, b: WatchedImage) => {
            return a.name.localeCompare(b.name);
        })
        .map((image) => (
            <li className="flex border-b last:border-0 border-base-300 justify-between py-2">
                <span>{image.name}</span>
                <CondensedAlertButton type="button" onClick={getRemoveFromWatch(image.name)}>
                    <XCircle className="h-3 w-3 mr-1" />
                    Remove watch
                </CondensedAlertButton>
            </li>
        ));

    return (
        <CustomDialogue
            className="w-3/4 md:w-1/2 lg:w-1/3"
            title="Manage Watched Images"
            text=""
            cancelText="Return to Image List"
            onCancel={closeDialog}
        >
            <div className="border-b border-base-300 ">
                <Formik
                    initialValues={{ imageTag: '' }}
                    validationSchema={Yup.object({
                        imageTag: Yup.string()
                            .matches(
                                /([a-z.]+\/)([a-z0-9-]+\/)([a-z0-9-@./]+)?(?::[0-9a-z\-.]+)?/,
                                'Must be a valid path to a container image'
                            )
                            .required('Required'),
                    })}
                    onSubmit={addToWatch}
                    // }}
                >
                    {({ getFieldProps, errors, touched, handleSubmit, isSubmitting }) => (
                        <form
                            className="flex flex-col leading-normal p-4 text-base-600 w-full"
                            onSubmit={handleSubmit}
                        >
                            <p className="mb-2">
                                Enter an image name to mark it as watched, so that it will continue
                                to be scanned even if no deployments use it.
                            </p>
                            <div className={wrapperMarginClassName}>
                                <label htmlFor="name" className={labelClassName}>
                                    <span>Image Name</span>
                                    <br />
                                    <span className={sublabelClassName}>
                                        The fully-qualified image name, beginning with the registry,
                                        and ending with the tag
                                    </span>
                                </label>
                                <div data-testid="input-wrapper">
                                    <input
                                        type="text"
                                        id="imageTag"
                                        {...getFieldProps('imageTag')}
                                        className={inputTextClassName}
                                    />
                                </div>
                                {touched.imageTag && errors.imageTag ? (
                                    <div className="bg-alert-300 p-2 text-alert-800 text-xs rounded-b border-alert-400 border-b border-l border-r flex flex-col">
                                        <ErrorMessage name="imageTag" />
                                    </div>
                                ) : null}
                            </div>
                            <div className="flex flex-row items-center justify-between">
                                <Button colorType="success" type="submit" isDisabled={isSubmitting}>
                                    {isSubmitting && (
                                        <span className="pr-2">
                                            <ClipLoader loading size="16" color="currentColor" />
                                        </span>
                                    )}
                                    <span>
                                        {isSubmitting
                                            ? 'Adding image—this may take some time...'
                                            : 'Add Image'}
                                    </span>
                                </Button>
                            </div>
                        </form>
                    )}
                </Formik>
                {!!successMessage && (
                    <div className="px-4 py-2">
                        <Message type="success">{successMessage}</Message>
                    </div>
                )}
                {!!errorMessage && (
                    <div className="px-4 py-2">
                        <Message type="error">{errorMessage}</Message>
                    </div>
                )}
                <div
                    className="flex flex-col leading-normal p-4 text-base-600 w-full overflow-y-auto"
                    style={{ maxHeight: '200px' }}
                >
                    <h3 className="font-700 mb-2">Images Currently Being Watched</h3>
                    {currentWatchedImages.length > 0 ? (
                        <ol>{imageList}</ol>
                    ) : (
                        <p className="text-center py-2">No images are currently being watched.</p>
                    )}
                </div>
            </div>
        </CustomDialogue>
    );
};

export default WatchedImagesDialog;
