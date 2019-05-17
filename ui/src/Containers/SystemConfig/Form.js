import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { reduxForm } from 'redux-form';

import ConfigFormWidget from './ConfigFormWidget';

const Form = ({ initialValues, onSubmit }) => (
    <>
        <form
            className="flex flex-col justify-between md:flex-row overflow-auto px-2 w-full"
            initialvalues={initialValues}
            onSubmit={onSubmit}
        >
            <ConfigFormWidget type="header" />
            <ConfigFormWidget type="footer" />
        </form>
    </>
);

Form.propTypes = {
    onSubmit: PropTypes.func.isRequired,
    initialValues: PropTypes.shape({})
};

Form.defaultProps = {
    initialValues: null
};

export default reduxForm({
    form: 'system-config-form'
})(
    connect(
        null,
        null
    )(Form)
);
