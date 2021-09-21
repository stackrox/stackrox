import React from 'react';
import PropTypes from 'prop-types';
import { useFormik } from 'formik';
import { object, string } from 'yup';
import { TextArea, ActionGroup, Button, Form, FormGroup } from '@patternfly/react-core';

const CommentForm = ({ initialFormValues, onSubmit }) => {
    const { errors, values, handleChange, handleBlur } = useFormik({
        initialValues: initialFormValues,
        validationSchema: object().shape({
            message: string().trim().required('This field is required'),
        }),
    });
    function onChange(_value, event) {
        handleChange(event);
    }
    function handleSubmit() {
        onSubmit(values);
    }
    const validatedState = errors.message ? 'error' : 'default';
    return (
        <Form>
            <FormGroup validated={validatedState} helperTextInvalid={errors.message}>
                <TextArea
                    data-testid="comment-textarea"
                    name="message"
                    rows="5"
                    placeholder="Write a comment here..."
                    onChange={onChange}
                    onBlur={handleBlur}
                    value={values.message}
                    // eslint-disable-next-line jsx-a11y/no-autofocus
                    autoFocus
                    aria-label="Comment Input"
                    validated={validatedState}
                />
            </FormGroup>
            <ActionGroup>
                <Button
                    variant="primary"
                    data-testid="save-comment-button"
                    isDisabled={values.message === '' || validatedState === 'error'}
                    onClick={handleSubmit}
                >
                    Save
                </Button>
            </ActionGroup>
        </Form>
    );
};

CommentForm.propTypes = {
    initialFormValues: PropTypes.shape({}),
    onSubmit: PropTypes.func.isRequired,
};

CommentForm.defaultProps = {
    initialFormValues: {},
};

export default CommentForm;
