import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { reduxForm, Field } from 'redux-form';

class SimpleForm extends Component {
    static propTypes = {
        handleSubmit: PropTypes.func.isRequired,
        fields: PropTypes.arrayOf(
            PropTypes.shape({
                label: PropTypes.string,
                value: PropTypes.string,
                placeholder: PropTypes.string
            })
        ).isRequired
    };

    renderFields = () => {
        const fields = this.props.fields.map(field => (
            <div key={field.label}>
                <div className="py-2 text-primary-500">{field.label}</div>
                <Field
                    name={field.value}
                    component="input"
                    type="text"
                    className="border rounded-l p-3 border-base-300 font-400"
                    autoComplete=""
                    placeholder={field.placeholder}
                />
            </div>
        ));
        return <div>{fields}</div>;
    };

    render() {
        return (
            <form onSubmit={this.props.handleSubmit} className="p-4">
                {this.renderFields()}
            </form>
        );
    }
}

export default reduxForm({ form: 'simpleform' })(SimpleForm);
