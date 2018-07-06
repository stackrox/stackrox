import React, { Component } from 'react';
import PropTypes from 'prop-types';

import FormField from 'Containers/Integrations/FormField';

class FormFields extends Component {
    static propTypes = {
        formApi: PropTypes.shape({
            setValue: PropTypes.func.isRequired,
            values: PropTypes.object.isRequired
        }).isRequired,
        fields: PropTypes.arrayOf(
            PropTypes.shape({
                label: PropTypes.string.isRequired,
                key: PropTypes.string.isRequired,
                type: PropTypes.string.isRequired,
                placeholder: PropTypes.string,
                options: PropTypes.arrayOf(PropTypes.object)
            })
        ).isRequired
    };

    render() {
        return this.props.fields.map(field => (
            <label className="flex mt-4" htmlFor={field.key} key={field.label}>
                <div className="mr-4 flex items-center w-2/3 capitalize">{field.label}</div>
                <FormField formApi={this.props.formApi} field={field} />
            </label>
        ));
    }
}

export default FormFields;
