import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { reduxForm } from 'redux-form';

import ReduxTextAreaField from 'Components/forms/ReduxTextAreaField';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxToggleField from 'Components/forms/ReduxToggleField';
import ReduxColorPickerField from 'Components/forms/ReduxColorPickerField';

const backgroundSizeOptions = [
    {
        label: 'Small',
        value: 'small'
    },
    {
        label: 'Medium',
        value: 'medium'
    },
    {
        label: 'Large',
        value: 'large'
    }
];

const keyClassName = 'py-2 text-base-600 font-700 capitalize';

const ConfigFormWidget = ({ type }) => (
    <div className="px-3 pt-5 w-full">
        <div className="bg-base-100 border-base-200 shadow">
            <div className="py-2 px-4 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between items-center">
                {`${type} configuration`}
                <ReduxToggleField name={type} />
            </div>

            <div className="flex flex-col pt-2 pb-4 px-4 w-full">
                <div className="flex w-full justify-between">
                    <div className="w-full pr-4">
                        <div className={keyClassName}>Text (2000 character limit):</div>
                        <ReduxTextAreaField
                            name={`${type}Text`}
                            placeholder={`Place ${type} text here...`}
                            maxlength="2000"
                        />
                    </div>
                    <div className="w-1/6">
                        <div className={keyClassName}>Text Color:</div>
                        <ReduxColorPickerField name={`${type}TextColor`} />
                    </div>
                </div>
                <div className="border-base-300 border-t flex justify-between mt-6 pt-4 w-full">
                    <div className="w-full pr-4">
                        <div className={keyClassName}>{`${type} Size:`}</div>
                        <ReduxSelectField name={`${type}Size`} options={backgroundSizeOptions} />
                    </div>
                    <div className="w-1/6">
                        <div className={keyClassName}>Bg Color:</div>
                        <ReduxColorPickerField name={`${type}BackgroundColor`} />
                    </div>
                </div>
            </div>
        </div>
    </div>
);

ConfigFormWidget.propTypes = {
    type: PropTypes.string.isRequired
};

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
    // handleSubmit: PropTypes.func.isRequired,
    // onSubmit: PropTypes.func.isRequired,
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
