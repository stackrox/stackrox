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
                placeholder: PropTypes.string,
                type: PropTypes.string,
                options: PropTypes.string
            })
        ).isRequired
    };

    renderTextField = field => (
        <Field
            name={field.value}
            component="input"
            type="text"
            className={`border rounded-l p-3 border-base-300 w-full font-400 ${
                field.disabled ? 'bg-base-100' : ''
            }`}
            disabled={field.disabled}
            autoComplete=""
            placeholder={field.placeholder}
        />
    );

    renderCheckboxField = field => (
        <Field name={field.value} component="input" type="checkbox" disabled={field.disabled} />
    );

    renderField = field => {
        switch (field.type) {
            case 'text':
                return this.renderTextField(field);
            case 'checkbox':
                return this.renderCheckboxField(field);
            default:
                return null;
        }
    };

    renderFields = () => {
        const fields = this.props.fields.map(field => (
            <div className="py-2" key={field.label}>
                <div className="py-2 text-primary-500">{field.label}</div>
                {this.renderField(field)}
            </div>
        ));
        return <div>{fields}</div>;
    };

    render() {
        return (
            <form onSubmit={this.props.handleSubmit} className="p-4 w-full mb-8">
                {this.renderFields()}
            </form>
        );
    }
}

export default reduxForm({ form: 'simpleform' })(SimpleForm);
