import React, { Component } from 'react';
import PropTypes from 'prop-types';

import formDescriptors from 'Containers/Integrations/formDescriptors';
import FormField from 'Containers/Integrations/FormField';

class FormFields extends Component {
    static propTypes = {
        formApi: PropTypes.shape({
            setValue: PropTypes.func.isRequired,
            values: PropTypes.object.isRequired
        }).isRequired,

        source: PropTypes.oneOf(['imageIntegrations', 'notifiers', 'authProviders', 'clusters'])
            .isRequired,
        type: PropTypes.string.isRequired
    };

    render() {
        const fields = formDescriptors[this.props.source][this.props.type];
        return fields.map(field => (
            <label className="flex mt-4" htmlFor={field.key} key={field.label}>
                <div className="mr-4 flex items-center w-2/3 capitalize">{field.label}</div>
                <FormField formApi={this.props.formApi} field={field} />
            </label>
        ));
    }
}

export default FormFields;
