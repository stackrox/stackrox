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
    ModalBody,
    ModalFooter,
    ModalHeader,
    TextInput,
    Title,
} from '@patternfly/react-core';

import useAnalytics, {
    BASE_IMAGE_REFERENCE_ADD_SUBMITTED,
    BASE_IMAGE_REFERENCE_ADD_SUCCESS,
    BASE_IMAGE_REFERENCE_ADD_FAILURE,
} from 'hooks/useAnalytics';
import { addBaseImage, updateBaseImageTagPattern } from 'services/BaseImagesService';
import type { BaseImageReference } from 'services/BaseImagesService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useRestMutation from 'hooks/useRestMutation';

/**
 * Categorizes API errors into meaningful error types for analytics tracking.
 */
function categorizeErrorType(error: unknown): string {
    const errorMessage = getAxiosErrorMessage(error).toLowerCase();

    if (errorMessage.includes('invalid') || errorMessage.includes('format')) {
        return 'INVALID_IMAGE_NAME';
    }
    if (errorMessage.includes('duplicate') || errorMessage.includes('already exists')) {
        return 'DUPLICATE_ENTRY';
    }
    if (errorMessage.includes('integration') || errorMessage.includes('registry')) {
        return 'NO_VALID_INTEGRATION';
    }
    if (errorMessage.includes('network') || errorMessage.includes('timeout')) {
        return 'NETWORK_ERROR';
    }
    return 'UNKNOWN';
}

export type BaseImagesModalProps = {
    isOpen: boolean;
    onClose: () => void;
    onSuccess: () => void;
    baseImageToEdit?: BaseImageReference | null;
};

const addValidationSchema = yup.object({
    baseImagePath: yup
        .string()
        .required('Base image path is required')
        .test(
            'has-tag',
            'Base image path must include a tag mask after ":" - for a registry with a port use "registry:port/repo:tag" (e.g. "registry:5000/repo:1.*")',
            (value) => {
                if (!value) {
                    return false;
                }
                const { tagPattern } = parseBaseImagePath(value);
                return tagPattern.length > 0;
            }
        ),
});

const editValidationSchema = yup.object({
    baseImageTagPattern: yup.string().required('Tag pattern is required'),
});

type AddFormData = yup.InferType<typeof addValidationSchema>;
type EditFormData = yup.InferType<typeof editValidationSchema>;

/**
 * Parses a base image path into repository path and tag pattern.
 * Only a colon in the final path segment separates the tag; a colon before the last
 * "/" belongs to a registry port. Returns an empty tagPattern when no tag is present.
 * Examples:
 *   "docker.io/library/ubuntu:22.04"      -> { repoPath: "docker.io/library/ubuntu", tagPattern: "22.04" }
 *   "registry:5000/library/ubuntu:1.*"    -> { repoPath: "registry:5000/library/ubuntu", tagPattern: "1.*" }
 *   "registry:5000/library/ubuntu"        -> { repoPath: "registry:5000/library/ubuntu", tagPattern: "" }
 */
export function parseBaseImagePath(path: string): { repoPath: string; tagPattern: string } {
    const lastSlashIndex = path.lastIndexOf('/');
    const tagColonIndex = path.indexOf(':', lastSlashIndex + 1);
    if (tagColonIndex === -1) {
        return { repoPath: path, tagPattern: '' };
    }
    const repoPath = path.substring(0, tagColonIndex);
    const tagPattern = path.substring(tagColonIndex + 1);
    return { repoPath, tagPattern };
}

/**
 * Modal form for adding or editing a base image.
 * In add mode: user enters full path (repo:tag), parsed into components.
 * In edit mode: repo path shown read-only, only tag pattern is editable.
 */
function BaseImagesModal({
    isOpen,
    onClose,
    onSuccess,
    baseImageToEdit = null,
}: BaseImagesModalProps) {
    const { analyticsTrack } = useAnalytics();

    const isEditMode = baseImageToEdit !== null;

    const addMutation = useRestMutation(
        ({
            baseImageRepoPath,
            baseImageTagPattern,
        }: {
            baseImageRepoPath: string;
            baseImageTagPattern: string;
        }) => addBaseImage(baseImageRepoPath, baseImageTagPattern)
    );

    const updateMutation = useRestMutation(
        ({ id, baseImageTagPattern }: { id: string; baseImageTagPattern: string }) =>
            updateBaseImageTagPattern(id, baseImageTagPattern)
    );

    const activeMutation = isEditMode ? updateMutation : addMutation;

    // Add mode form
    const addFormik = useFormik<AddFormData>({
        initialValues: { baseImagePath: '' },
        validationSchema: addValidationSchema,
        validateOnMount: true,
        onSubmit: (formValues: AddFormData, { setSubmitting }: FormikHelpers<AddFormData>) => {
            analyticsTrack(BASE_IMAGE_REFERENCE_ADD_SUBMITTED);

            // Parse user input (e.g., "docker.io/library/ubuntu:22.04") into separate components
            const { repoPath, tagPattern } = parseBaseImagePath(formValues.baseImagePath);

            addMutation.mutate(
                { baseImageRepoPath: repoPath, baseImageTagPattern: tagPattern },
                {
                    onSuccess: () => {
                        analyticsTrack(BASE_IMAGE_REFERENCE_ADD_SUCCESS);
                        formik.resetForm();
                        onSuccess();
                    },
                    onError: (error) => {
                        analyticsTrack({
                            event: BASE_IMAGE_REFERENCE_ADD_FAILURE,
                            properties: { errorType: categorizeErrorType(error) },
                        });
                    },
                    onSettled: () => {
                        setSubmitting(false);
                    },
                }
            );
        },
    });

    // Edit mode form
    const editFormik = useFormik<EditFormData>({
        initialValues: { baseImageTagPattern: baseImageToEdit?.baseImageTagPattern ?? '' },
        validationSchema: editValidationSchema,
        validateOnMount: true,
        enableReinitialize: true,
        onSubmit: (formValues, { setSubmitting }: FormikHelpers<EditFormData>) => {
            if (!baseImageToEdit) {
                return;
            }
            updateMutation.mutate(
                { id: baseImageToEdit.id, baseImageTagPattern: formValues.baseImageTagPattern },
                {
                    onSuccess: () => {
                        editFormik.resetForm();
                        onSuccess();
                    },
                    onSettled: () => setSubmitting(false),
                }
            );
        },
    });

    const formik = isEditMode ? editFormik : addFormik;
    const isSubmitting = formik.isSubmitting || activeMutation.isLoading;

    const handleModalClose = () => {
        if (!isSubmitting) {
            formik.resetForm();
            activeMutation.reset();
            onClose();
        }
    };

    const modalTitle = isEditMode ? 'Edit base image tag pattern' : 'Add base image path';
    const successMessage = isEditMode
        ? 'Base image tag pattern successfully updated'
        : 'Base image successfully added';
    const errorTitle = isEditMode ? 'Error updating base image' : 'Error adding base image';

    return (
        <Modal
            aria-labelledby="base-image-modal-title"
            isOpen={isOpen}
            onClose={handleModalClose}
            variant="medium"
        >
            <ModalHeader>
                <Title id="base-image-modal-title" headingLevel="h2">
                    {modalTitle}
                </Title>
            </ModalHeader>
            <ModalBody>
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    {activeMutation.isSuccess && (
                        <Alert variant="success" isInline title={successMessage} component="p" />
                    )}
                    {activeMutation.isError && (
                        <Alert variant="danger" isInline title={errorTitle} component="p">
                            {getAxiosErrorMessage(activeMutation.error)}
                        </Alert>
                    )}
                    {isEditMode ? (
                        <Form onSubmit={editFormik.handleSubmit}>
                            <FormGroup label="Repository path" fieldId="baseImageRepoPath">
                                <TextInput
                                    id="baseImageRepoPath"
                                    type="text"
                                    value={baseImageToEdit.baseImageRepoPath}
                                    isDisabled
                                    aria-label="Repository path (read-only)"
                                />
                                <FormHelperText>
                                    <HelperText>
                                        <HelperTextItem>
                                            Repository path cannot be changed after creation
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            </FormGroup>
                            <FormGroup label="Tag pattern" fieldId="baseImageTagPattern" isRequired>
                                <TextInput
                                    id="baseImageTagPattern"
                                    type="text"
                                    value={editFormik.values.baseImageTagPattern}
                                    validated={
                                        editFormik.errors.baseImageTagPattern &&
                                        editFormik.touched.baseImageTagPattern
                                            ? 'error'
                                            : 'default'
                                    }
                                    onChange={(e) => editFormik.handleChange(e)}
                                    onBlur={editFormik.handleBlur}
                                    isDisabled={isSubmitting}
                                    placeholder="22.04 or 3.*"
                                    isRequired
                                />
                                <FormHelperText>
                                    <HelperText>
                                        {editFormik.errors.baseImageTagPattern &&
                                            editFormik.touched.baseImageTagPattern && (
                                                <HelperTextItem variant="error">
                                                    {editFormik.errors.baseImageTagPattern}
                                                </HelperTextItem>
                                            )}
                                        <HelperTextItem>
                                            Tag can be a specific version or pattern (e.g., 1.*)
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            </FormGroup>
                        </Form>
                    ) : (
                        <Form onSubmit={addFormik.handleSubmit}>
                            <FormGroup label="Base image path" fieldId="baseImagePath" isRequired>
                                <TextInput
                                    id="baseImagePath"
                                    type="text"
                                    value={addFormik.values.baseImagePath}
                                    validated={
                                        addFormik.errors.baseImagePath &&
                                        addFormik.touched.baseImagePath
                                            ? 'error'
                                            : 'default'
                                    }
                                    onChange={(e) => addFormik.handleChange(e)}
                                    onBlur={addFormik.handleBlur}
                                    isDisabled={isSubmitting}
                                    placeholder="example-registry.io/path/to/image:tag"
                                    isRequired
                                />
                                <FormHelperText>
                                    <HelperText>
                                        {addFormik.errors.baseImagePath &&
                                            addFormik.touched.baseImagePath && (
                                                <HelperTextItem variant="error">
                                                    {addFormik.errors.baseImagePath}
                                                </HelperTextItem>
                                            )}
                                        <HelperTextItem>
                                            Include repository path and tag (e.g.,
                                            example-registry.io/path/to/image:tag). Tag can be a
                                            pattern (e.g., 1.*)
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            </FormGroup>
                        </Form>
                    )}
                </Flex>
            </ModalBody>
            <ModalFooter>
                <Button
                    variant="primary"
                    onClick={formik.submitForm}
                    isLoading={isSubmitting}
                    isDisabled={isSubmitting || !formik.isValid}
                >
                    Save
                </Button>
                <Button variant="link" onClick={handleModalClose} isDisabled={isSubmitting}>
                    Cancel
                </Button>
            </ModalFooter>
        </Modal>
    );
}

export default BaseImagesModal;
