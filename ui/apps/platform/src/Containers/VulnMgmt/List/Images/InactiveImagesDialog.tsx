/* eslint-disable jsx-a11y/control-has-associated-label */
import React, { ReactElement, useState } from 'react';
import { Formik, ErrorMessage } from 'formik';
import * as Yup from 'yup';
import { ClipLoader } from 'react-spinners';
import { Button, Message } from '@stackrox/ui-components';

import CustomDialogue from 'Components/CustomDialogue';
import {
    labelClassName,
    sublabelClassName,
    wrapperMarginClassName,
    inputTextClassName,
} from 'constants/form.constants';
import { watchImage } from 'services/ImagesService';

type InactiveImagesDialogProps = {
    closeDialog: () => void;
};

const InactiveImagesDialog = ({ closeDialog }: InactiveImagesDialogProps): ReactElement => {
    const [successMessage, setSuccessMessage] = useState<ReactElement | string>('');
    const [errorMessage, setErrorMessage] = useState<ReactElement | string>('');

    return (
        <CustomDialogue
            className="max-w-3/4 md:max-w-2/3 lg:max-w-1/2 min-w-1/2 md:min-w-1/3 lg:min-w-1/4"
            title="Manage Inactive Images"
            text=""
            cancelText="Return to Image List"
            onCancel={closeDialog}
        >
            <Formik
                initialValues={{ imageTag: '' }}
                validationSchema={Yup.object({
                    imageTag: Yup.string()
                        .matches(
                            /(?:[a-z.]+\/)([a-z/]+)+(?::[0-9]+)?/,
                            'Must be a valid path to a container image'
                        )
                        .required('Required'),
                })}
                // eslint-disable-next-line react/jsx-no-bind
                onSubmit={(values, { setSubmitting }) => {
                    setSuccessMessage('');
                    setErrorMessage('');
                    watchImage(values.imageTag)
                        .then((image) => {
                            setSuccessMessage(
                                <div>
                                    <strong>{image?.normalizedName}</strong> has been added to the
                                    list of images to be scanned.
                                </div>
                            );
                        })
                        .catch((error) => {
                            setErrorMessage(
                                <div>
                                    <p className="mb-2">
                                        The image name you submitted,{' '}
                                        <strong>{values.imageTag}</strong> could not be processed.
                                    </p>
                                    <p className="mb-1">
                                        The server response with the following message.
                                    </p>
                                    <blockquote className="p-4 border-l-4 quote">
                                        {error.message}
                                    </blockquote>
                                </div>
                            );
                        })
                        .finally(() => {
                            setSubmitting(false);
                        });
                }}
            >
                {({ getFieldProps, errors, touched, handleSubmit, isSubmitting }) => (
                    <form
                        className="border-b border-base-300 flex flex-col leading-normal p-4 text-base-600 w-full"
                        onSubmit={handleSubmit}
                    >
                        <div className={wrapperMarginClassName}>
                            <label htmlFor="name" className={labelClassName}>
                                <span>Image Name</span>
                                <br />
                                <span className={sublabelClassName}>
                                    The fully-qualified image name, beginning with the registry, and
                                    ending with the tag
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
                        {!!successMessage && (
                            <div className="py-2">
                                <Message type="success">{successMessage}</Message>
                            </div>
                        )}
                        {!!errorMessage && (
                            <div className="py-2">
                                <Message type="error">{errorMessage}</Message>
                            </div>
                        )}
                    </form>
                )}
            </Formik>
        </CustomDialogue>
    );
};

export default InactiveImagesDialog;
