import React from 'react';
import PropTypes from 'prop-types';

import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxTextAreaField from 'Components/forms/ReduxTextAreaField';
import ReduxCheckboxField from 'Components/forms/ReduxCheckboxField';
import ReduxMultiSelectField from 'Components/forms/ReduxMultiSelectField';
import ReduxMultiSelectCreatableField from 'Components/forms/ReduxMultiSelectCreatableField';
import ReduxNumericInputField from 'Components/forms/ReduxNumericInputField';
import ReduxToggleField from 'Components/forms/ReduxToggleField';

export default function Field({ field }) {
    if (field === undefined) return null;
    switch (field.type) {
        case 'text':
            return (
                <ReduxTextField
                    key={field.jsonpath}
                    name={field.jsonpath}
                    disabled={field.disabled}
                    placeholder={field.placeholder}
                />
            );
        case 'checkbox':
            return <ReduxCheckboxField name={field.jsonpath} disabled={field.disabled} />;
        case 'toggle':
            return (
                <ReduxToggleField
                    name={field.jsonpath}
                    disabled={field.disabled}
                    reverse={field.reverse}
                />
            );
        case 'select':
            return (
                <ReduxSelectField
                    key={field.jsonpath}
                    name={field.jsonpath}
                    options={field.options}
                    placeholder={field.placeholder}
                    disabled={field.disabled}
                    defaultValue={field.default}
                />
            );
        case 'multiselect':
            return <ReduxMultiSelectField name={field.jsonpath} options={field.options} />;
        case 'multiselect-creatable':
            return <ReduxMultiSelectCreatableField name={field.jsonpath} options={field.options} />;
        case 'textarea':
            return (
                <ReduxTextAreaField
                    name={field.jsonpath}
                    disabled={field.disabled}
                    placeholder={field.placeholder}
                />
            );
        case 'number':
            return (
                <ReduxNumericInputField
                    key={field.jsonpath}
                    name={field.jsonpath}
                    min={field.min}
                    max={field.max}
                    step={field.step}
                    placeholder={field.placeholder}
                />
            );
        case 'group':
            return field.jsonpaths.map(input => <Field field={input} />);
        default:
            throw new Error(`Unknown field type: ${field.type}`);
    }
}

Field.propsTypes = {
    field: PropTypes.shape({
        type: PropTypes.string.isRequired
    }).isRequired
};
