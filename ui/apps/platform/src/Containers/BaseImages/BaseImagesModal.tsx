import { useFormik } from 'formik';
import type { FormikHelpers } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    Button,
    Flex,
    Form,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    Modal,
    TextInput,
    Title,
} from '@patternfly/react-core';

import { addBaseImage } from 'services/BaseImagesService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useRestMutation from 'hooks/useRestMutation';

export type BaseImagesModalProps = {
    isOpen: boolean;
    onClose: () => void;
    onSuccess?: () => void;
};

const validationSchema = yup.object({
    baseImagePath: yup
        .string()
        .required('Base image path is required')
        .test(
            'has-colon',
            'Base image path must include both repository and tag separated by ":"',
            (value) => {
                if (!value?.includes(':')) {
                    return false;
                }
                const lastColonIndex = value.lastIndexOf(':');
                const tagPattern = value.substring(lastColonIndex + 1);
                // Tag pattern must not be empty
                return tagPattern.length > 0;
            }
        ),
});

type FormData = yup.InferType<typeof validationSchema>;

function parseBaseImagePath(path: string): { repoPath: string; tagPattern: string } {
    const lastColonIndex = path.lastIndexOf(':');
    const repoPath = path.substring(0, lastColonIndex);
    const tagPattern = path.substring(lastColonIndex + 1);
    return { repoPath, tagPattern };
}

function BaseImagesModal({ isOpen, onClose, onSuccess }: BaseImagesModalProps) {
    const addBaseImageMutation = useRestMutation(
        ({
            baseImageRepoPath,
            baseImageTagPattern,
        }: {
            baseImageRepoPath: string;
            baseImageTagPattern: string;
        }) => addBaseImage(baseImageRepoPath, baseImageTagPattern)
    );

    const formik = useFormik({
        initialValues: { baseImagePath: '' },
        validationSchema,
        onSubmit: (formValues: FormData, { setSubmitting }: FormikHelpers<FormData>) => {
            const { repoPath, tagPattern } = parseBaseImagePath(formValues.baseImagePath);

            addBaseImageMutation.mutate(
                {
                    baseImageRepoPath: repoPath,
                    baseImageTagPattern: tagPattern,
                },
                {
                    onSuccess: () => {
                        formik.resetForm();
                        onSuccess?.();
                    },
                    onSettled: () => {
                        setSubmitting(false);
                    },
                }
            );
        },
    });

    const isBaseImagePathFieldInvalid = !!(
        formik.errors.baseImagePath && formik.touched.baseImagePath
    );
    const baseImagePathFieldValidated = isBaseImagePathFieldInvalid ? 'error' : 'default';
    const isSubmitting = formik.isSubmitting || addBaseImageMutation.isLoading;

    const handleModalClose = () => {
        if (!isSubmitting) {
            formik.resetForm();
            addBaseImageMutation.reset();
            onClose();
        }
    };

    return (
        <Modal
            aria-labelledby="add-base-image-modal-title"
            header={
                <Title id="add-base-image-modal-title" headingLevel="h2">
                    Add base image path
                </Title>
            }
            isOpen={isOpen}
            onClose={handleModalClose}
            variant="medium"
            showClose
            actions={[
                <Button
                    key="save"
                    variant="primary"
                    onClick={formik.submitForm}
                    isLoading={isSubmitting}
                    isDisabled={isSubmitting || !formik.isValid}
                >
                    Save
                </Button>,
                <Button
                    key="cancel"
                    variant="link"
                    onClick={handleModalClose}
                    isDisabled={isSubmitting}
                >
                    Cancel
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                {addBaseImageMutation.isSuccess && (
                    <Alert
                        variant="success"
                        isInline
                        title="Base image successfully added"
                        component="p"
                    />
                )}
                {addBaseImageMutation.isError && (
                    <Alert variant="danger" isInline title="Error adding base image" component="p">
                        {getAxiosErrorMessage(addBaseImageMutation.error)}
                    </Alert>
                )}
                <Form onSubmit={formik.handleSubmit}>
                    <FormGroup label="Base image path" fieldId="baseImagePath" isRequired>
                        <TextInput
                            id="baseImagePath"
                            type="text"
                            value={formik.values.baseImagePath}
                            validated={baseImagePathFieldValidated}
                            onChange={(e) => formik.handleChange(e)}
                            onBlur={formik.handleBlur}
                            isDisabled={isSubmitting}
                            placeholder="Example: docker.io/library/ubuntu:22.04"
                            isRequired
                        />
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem variant={baseImagePathFieldValidated}>
                                    {isBaseImagePathFieldInvalid
                                        ? formik.errors.baseImagePath
                                        : 'Include repository path and tag (e.g., docker.io/library/ubuntu:22.04). Tag can be a pattern (e.g., v*.*)'}
                                </HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    </FormGroup>
                </Form>
            </Flex>
        </Modal>
    );
}

export default BaseImagesModal;
