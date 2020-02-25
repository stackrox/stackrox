import React from 'react';
import PropTypes from 'prop-types';

const TextArea = ({ name, required, register, errors, ...rest }) => {
    return (
        <>
            <textarea
                className="border border-base-400 leading-normal p-1 w-full"
                name={name}
                ref={register({ required })}
                {...rest}
            />
            {errors[name] && <span className="text-alert-700">This field is required</span>}
        </>
    );
};

TextArea.propTypes = {
    name: PropTypes.string.isRequired, // This is the key the input will use to identify itself and grab the appropriate errors to display
    required: PropTypes.bool,
    register: PropTypes.func.isRequired,
    errors: PropTypes.shape({}).isRequired
};

TextArea.defaultProps = {
    required: false
};

export default TextArea;
