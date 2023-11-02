import React from 'react';
import { Button, Form, FormGroup, TextInput } from '@patternfly/react-core';
import { FormikHelpers, useFormik } from 'formik';
import * as yup from 'yup';

import { UseRestQueryReturn } from 'hooks/useRestQuery';
import { UseRestMutationReturn } from 'hooks/useRestMutation';
import useAnalytics, { WATCH_IMAGE_SUBMITTED } from 'hooks/useAnalytics';
import { WatchedImage } from 'types/image.proto';

const validationSchema = yup.object({
    imageName: yup.string().required('A valid image name is required'),
});

type FormData = yup.InferType<typeof validationSchema>;

export type WatchedImagesFormProps = {
    defaultWatchedImageName: string;
    watchedImagesRequest: UseRestQueryReturn<WatchedImage[]>;
    watchImage: UseRestMutationReturn<string, string>['mutate'];
};

function WatchedImagesForm({
    defaultWatchedImageName,
    watchedImagesRequest,
    watchImage,
}: WatchedImagesFormProps) {
    const {
        values,
        errors,
        touched,
        handleChange,
        handleBlur,
        handleSubmit,
        submitForm,
        isSubmitting,
    } = useFormik({
        initialValues: { imageName: defaultWatchedImageName },
        validationSchema,
        onSubmit: addToWatchedImages,
    });
    const isNameFieldInvalid = !!(errors.imageName && touched.imageName);
    const nameFieldValidated = isNameFieldInvalid ? 'error' : 'default';

    const { analyticsTrack } = useAnalytics();

    function addToWatchedImages(formValues: FormData, { setSubmitting }: FormikHelpers<FormData>) {
        analyticsTrack(WATCH_IMAGE_SUBMITTED);
        watchImage(formValues.imageName, {
            onSuccess: () => watchedImagesRequest.refetch(),
            onSettled: () => setSubmitting(false),
        });
    }

    return (
        <Form onSubmit={handleSubmit}>
            <FormGroup
                label="Image name"
                fieldId="imageName"
                isRequired
                validated={nameFieldValidated}
                helperText="The fully-qualified image name, beginning with the registry, and ending with the tag."
                helperTextInvalid={errors.imageName}
            >
                <TextInput
                    id="imageName"
                    type="text"
                    value={values.imageName}
                    validated={nameFieldValidated}
                    onChange={(_, e) => handleChange(e)}
                    onBlur={handleBlur}
                    isDisabled={isSubmitting}
                    placeholder="registry.example.com/namespace/image-name:tag"
                    isRequired
                />
            </FormGroup>
            <div>
                <Button
                    variant="secondary"
                    onClick={submitForm}
                    isDisabled={isSubmitting || isNameFieldInvalid}
                    isLoading={isSubmitting}
                >
                    Add image to watch list
                </Button>
            </div>
        </Form>
    );
}

export default WatchedImagesForm;
