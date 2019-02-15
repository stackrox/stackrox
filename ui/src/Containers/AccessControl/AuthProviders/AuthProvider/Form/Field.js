import React from 'react';
import PropTypes from 'prop-types';

import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxSelectCreatableField from 'Components/forms/ReduxSelectCreatableField';
import ReduxTextAreaField from 'Components/forms/ReduxTextAreaField';

const Field = props => {
    const { label, jsonPath, placeholder, type, options, html, customComponents, styles } = props;
    let field = null;
    switch (type) {
        case 'text':
            field = <ReduxTextField name={jsonPath} placeholder={placeholder} />;
            break;
        case 'select':
            field = (
                <ReduxSelectField
                    name={jsonPath}
                    options={options}
                    placeholder={placeholder}
                    customComponents={customComponents}
                    styles={styles}
                />
            );
            break;
        case 'selectcreatable':
            field = (
                <ReduxSelectCreatableField
                    name={jsonPath}
                    options={options}
                    placeholder={placeholder}
                    styles={styles}
                />
            );
            break;
        case 'textarea':
            field = <ReduxTextAreaField name={jsonPath} placeholder={placeholder} />;
            break;
        case 'html':
            return <div className="w-full mb-5">{html}</div>;
        default:
            field = null;
            break;
    }
    return (
        <div className="mb-4">
            <div className="py-1 text-base-600 font-700">{label}</div>
            <div className="w-full py-1">{field}</div>
        </div>
    );
};

Field.propTypes = {
    label: PropTypes.string,
    jsonPath: PropTypes.string,
    placeholder: PropTypes.string,
    type: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(
        PropTypes.shape({
            label: PropTypes.string,
            value: PropTypes.string
        })
    ),
    html: PropTypes.element,
    customComponents: PropTypes.shape({}),
    styles: PropTypes.shape({})
};

Field.defaultProps = {
    label: '',
    jsonPath: '',
    placeholder: '',
    options: [],
    html: <div />,
    customComponents: {},
    styles: {}
};

export default Field;
