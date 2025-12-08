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

/**
 * Parses a base image path into repository path and tag pattern.
 * Format: "docker.io/library/ubuntu:22.04" -> { repoPath: "docker.io/library/ubuntu", tagPattern: "22.04" }
 */
function parseBaseImagePath(path: string): { repoPath: string; tagPattern: string } {
    const lastColonIndex = path.lastIndexOf(':');
    const repoPath = path.substring(0, lastColonIndex);
    const tagPattern = path.substring(lastColonIndex + 1);
    return { repoPath, tagPattern };
}

/**
 * Modal form for adding a new base image. Handles form validation and submission,
 * parsing the input path into repo path and tag pattern components.
 */
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
            // Parse user input (e.g., "docker.io/library/ubuntu:22.04") into separate components
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

    /**
     * Prevents closing the modal while a submission is in progress.
     * Cleans up form state when successfully closed.
     */
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
                            placeholder="example-registry.io/path/to/image:tag"
                            isRequired
                        />
                        <FormHelperText>
                            <HelperText>
                                {isBaseImagePathFieldInvalid && (
                                    <HelperTextItem variant="error">
                                        {formik.errors.baseImagePath}
                                    </HelperTextItem>
                                )}
                                <HelperTextItem>
                                    Include repository path and tag (e.g.,
                                    example-registry.io/path/to/image:tag). Tag can be a pattern
                                    (e.g., 1.*)
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
