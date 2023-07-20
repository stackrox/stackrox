import React, { ReactElement, useState, useEffect } from 'react';
import { Formik, ErrorMessage } from 'formik';
import * as Yup from 'yup';
import { Alert, Button, Flex, FlexItem } from '@patternfly/react-core';

import {
    labelClassName,
    sublabelClassName,
    wrapperMarginClassName,
    inputTextClassName,
} from 'constants/form.constants';
import { getWatchedImages, watchImage, unwatchImage } from 'services/imageService';
import { WatchedImage } from 'types/image.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import CustomDialogue from '../../Components/CustomDialogue';

type WatchedImagesDialogProps = {
    closeDialog: () => void;
};

const WatchedImagesDialog = ({ closeDialog }: WatchedImagesDialogProps): ReactElement => {
    const [currentWatchedImages, setCurrentWatchedImages] = useState<WatchedImage[]>([]);
    const [successTitle, setSuccessTitle] = useState('');
    const [successMessage, setSuccessMessage] = useState<ReactElement | string>('');
    const [errorTitle, setErrorTitle] = useState('');
    const [errorMessage, setErrorMessage] = useState<ReactElement | string>('');

    function refreshWatchList() {
        getWatchedImages()
            .then((images) => {
                setCurrentWatchedImages(images);
            })
            .catch((error) => {
                setErrorTitle('Unable to retrieve the current list of watched images');
                setErrorMessage(getAxiosErrorMessage(error));
            });
    }

    useEffect(() => {
        refreshWatchList();
    }, []);

    function addToWatch(values, { setSubmitting }) {
        setSuccessTitle('');
        setSuccessMessage('');
        setErrorTitle('');
        setErrorMessage('');
        watchImage(values.imageTag)
            .then((normalizedName) => {
                setSuccessTitle('Image added');
                setSuccessMessage(
                    <>
                        <div>{normalizedName}</div>
                        {normalizedName !== values.imageTag && (
                            <div>{` (normalized form of ${values.imageTag as string})`}</div>
                        )}
                    </>
                );
                refreshWatchList();
            })
            .catch((error) => {
                setErrorTitle('Unable to process image name');
                setErrorMessage(
                    <>
                        <p className="mb-2">{values.imageTag}</p>
                        <p>{getAxiosErrorMessage(error)}</p>
                    </>
                );
            })
            .finally(() => {
                setSubmitting(false);
            });
    }

    function getRemoveFromWatch(imageName) {
        return function removeFromWatch() {
            setSuccessTitle('');
            setSuccessMessage('');
            setErrorTitle('');
            setErrorMessage('');
            unwatchImage(imageName)
                .then(() => {
                    setSuccessTitle('Removed watch for image');
                    setSuccessMessage(imageName);
                    refreshWatchList();
                })
                .catch((error) => {
                    setErrorTitle('Unable to remove watch for image');
                    setErrorMessage(getAxiosErrorMessage(error));
                });
        };
    }

    const imageList = currentWatchedImages
        .sort((a: WatchedImage, b: WatchedImage) => {
            return a.name.localeCompare(b.name);
        })
        .map((image) => (
            <li className="flex border-b last:border-0 border-base-300 py-2">
                <Flex
                    className="pf-u-w-100"
                    alignItems={{ default: 'alignItemsCenter' }}
                    flexWrap={{ default: 'nowrap' }}
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                >
                    <FlexItem>{image.name}</FlexItem>
                    <FlexItem>
                        <Button variant="danger" isSmall onClick={getRemoveFromWatch(image.name)}>
                            Remove watch
                        </Button>
                    </FlexItem>
                </Flex>
            </li>
        ));

    return (
        <CustomDialogue
            className="w-1/2 xxl:w-1/3"
            title="Manage watched images"
            cancelText="Close"
            onCancel={closeDialog}
        >
            <div className="px-4">
                <Formik
                    initialValues={{ imageTag: '' }}
                    validationSchema={Yup.object({
                        imageTag: Yup.string().required('Required'),
                    })}
                    onSubmit={addToWatch}
                >
                    {({ getFieldProps, errors, touched, handleSubmit, isSubmitting }) => (
                        <form
                            className="flex flex-col leading-normal pb-4 text-base-600 w-full"
                            onSubmit={handleSubmit}
                        >
                            <p className="mb-2">
                                Enter an image name to mark it as watched, so that it will continue
                                to be scanned even if no deployments use it.
                            </p>
                            <div className={wrapperMarginClassName}>
                                <label htmlFor="imageTag" className={labelClassName}>
                                    <span>Image name</span>
                                    <br />
                                    <span className={sublabelClassName}>
                                        The fully-qualified image name, beginning with the registry,
                                        and ending with the tag
                                    </span>
                                </label>
                                <input
                                    type="text"
                                    id="imageTag"
                                    {...getFieldProps('imageTag')}
                                    className={inputTextClassName}
                                />
                                {touched.imageTag && errors.imageTag ? (
                                    <div className="bg-alert-300 p-2 text-alert-800 text-xs rounded-b border-alert-400 border-b border-l border-r flex flex-col">
                                        <ErrorMessage name="imageTag" />
                                    </div>
                                ) : null}
                            </div>
                            <div className="flex flex-row">
                                <Button
                                    variant="primary"
                                    type="submit"
                                    isDisabled={isSubmitting}
                                    isLoading={isSubmitting}
                                >
                                    Add image
                                </Button>
                            </div>
                        </form>
                    )}
                </Formik>
                {!!successMessage && (
                    <div className="pb-4">
                        <Alert isInline variant="success" component="h3" title={successTitle}>
                            {successMessage}
                        </Alert>
                    </div>
                )}
                {!!errorMessage && (
                    <div className="pb-4">
                        <Alert isInline variant="danger" component="h3" title={errorTitle}>
                            {errorMessage}
                        </Alert>
                    </div>
                )}
                <div
                    className="flex flex-col leading-normal text-base-600 w-full overflow-y-auto"
                    style={{ maxHeight: '200px' }}
                >
                    <h3 className="font-700 my-2">Images currently being watched</h3>
                    {currentWatchedImages.length > 0 ? (
                        <ol>{imageList}</ol>
                    ) : (
                        <p>No images are currently being watched.</p>
                    )}
                </div>
            </div>
        </CustomDialogue>
    );
};

export default WatchedImagesDialog;
