import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { reduxForm } from 'redux-form';

import ConfigBannerFormWidget from './ConfigBannerFormWidget';
import ConfigLoginFormWidget from './ConfigLoginFormWidget';
import { pageLayoutClassName } from './Page';

const Form = ({ initialValues, onSubmit }) => (
    <>
        <form className={pageLayoutClassName} initialvalues={initialValues} onSubmit={onSubmit}>
            <div className="flex flex-col justify-between md:flex-row w-full">
                <ConfigBannerFormWidget type="header" />
                <ConfigBannerFormWidget type="footer" />
            </div>
            <div className="px-3 pt-5 w-full">
                <ConfigLoginFormWidget />
            </div>
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
