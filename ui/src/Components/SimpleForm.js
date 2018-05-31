import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { reduxForm } from 'redux-form';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxCheckboxField from 'Components/forms/ReduxCheckboxField';

class SimpleForm extends Component {
    static propTypes = {
        id: PropTypes.string,
        handleSubmit: PropTypes.func.isRequired,
        fields: PropTypes.arrayOf(
            PropTypes.shape({
                label: PropTypes.string
            })
        ).isRequired
    };

    static defaultProps = {
        id: ''
    };

    renderField = field => {
        switch (field.type) {
            case 'text':
                return (
                    <ReduxTextField
                        name={field.jsonpath}
                        disabled={field.disabled}
                        placeholder={field.placeholder}
                    />
                );
            case 'checkbox':
                return <ReduxCheckboxField name={field.jsonpath} disabled={field.disabled} />;
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
        const formId = this.props.id ? `${this.props.id}-form` : '';
        return (
            <form onSubmit={this.props.handleSubmit} className={`${formId} p-4 w-full mb-8`}>
                {this.renderFields()}
            </form>
        );
    }
}

export default reduxForm({ form: 'simpleform' })(SimpleForm);
