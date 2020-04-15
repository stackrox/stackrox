import React from 'react';
import PropTypes from 'prop-types';
import { Formik } from 'formik';
import { object, string } from 'yup';

const CommentForm = ({ initialFormValues, onSubmit }) => {
    return (
        <Formik
            initialValues={initialFormValues}
            validationSchema={object().shape({
                message: string()
                    .trim()
                    .required('This field is required')
            })}
            onSubmit={onSubmit}
        >
            {({ values, errors, handleChange, handleBlur, handleSubmit }) => (
                <form onSubmit={handleSubmit}>
                    <textarea
                        data-testid="comment-textarea"
                        className="form-textarea bg-base-100 text-base border border-base-400 leading-normal p-1 w-full"
                        name="message"
                        rows="5"
                        cols="33"
                        placeholder="Write a comment here..."
                        onChange={handleChange}
                        onBlur={handleBlur}
                        value={values.message}
                        // eslint-disable-next-line jsx-a11y/no-autofocus
                        autoFocus
                        aria-label="Comment Input"
                    />
                    {errors && errors.message && (
                        <span className="text-alert-700">{errors.message}</span>
                    )}
                    <div className="flex justify-end">
                        <button
                            className="bg-success-300 border border-success-800 p-1 rounded-sm text-sm text-success-900 uppercase hover:bg-success-400 cursor-pointer"
                            type="submit"
                        >
                            Save
                        </button>
                    </div>
                </form>
            )}
        </Formik>
    );
};

CommentForm.propTypes = {
    initialFormValues: PropTypes.shape({}),
    onSubmit: PropTypes.func.isRequired
};

CommentForm.defaultProps = {
    initialFormValues: {}
};

export default CommentForm;
