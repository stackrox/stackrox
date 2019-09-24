import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { reduxForm } from 'redux-form';

import ConfigBannerFormWidget from './ConfigBannerFormWidget';
import ConfigDataRetentionFormWidget from './ConfigDataRetentionFormWidget';
import ConfigLoginFormWidget from './ConfigLoginFormWidget';
import { pageLayoutClassName } from './SystemConfig.constants';

const Form = ({ initialValues, onSubmit, config }) => (
    <>
        <form className={pageLayoutClassName} initialvalues={initialValues} onSubmit={onSubmit}>
            <div className="px-3 pb-5 w-full">
                <ConfigDataRetentionFormWidget
                    initialValues={initialValues}
                    privateConfig={config.privateConfig}
                />
            </div>
            <div className="flex flex-col justify-between md:flex-row pb-5 w-full">
                <ConfigBannerFormWidget type="header" />
                <ConfigBannerFormWidget type="footer" />
            </div>
            <div className="px-3 pb-5 w-full">
                <ConfigLoginFormWidget />
            </div>
        </form>
    </>
);

Form.propTypes = {
    config: PropTypes.shape({
        privateConfig: PropTypes.shape({
            alertConfig: PropTypes.shape({}),
            imageRetentionDurationDays: PropTypes.number
        })
    }).isRequired,
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
