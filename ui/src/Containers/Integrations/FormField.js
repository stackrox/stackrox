import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Text, Select, Checkbox } from 'react-form';
import MultiSelect from 'react-select';

class FormField extends Component {
    static propTypes = {
        formApi: PropTypes.shape({
            setValue: PropTypes.func.isRequired,
            values: PropTypes.object.isRequired
        }).isRequired,

        field: PropTypes.shape({
            label: PropTypes.string.isRequired,
            key: PropTypes.string.isRequired,
            type: PropTypes.string.isRequired,
            placeholder: PropTypes.string,
            options: PropTypes.arrayOf(PropTypes.object)
        }).isRequired
    };

    render() {
        const handleMultiSelectChange = () => newValue => {
            const values = newValue !== '' ? newValue.split(',') : [];
            this.props.formApi.setValue(this.props.field.key, values);
        };

        switch (this.props.field.type) {
            case 'checkbox':
                return <Checkbox field={this.props.field.key} name={this.props.field.key} />;
            case 'text':
                return (
                    <Text
                        type="text"
                        className="border rounded w-full p-3 border-base-300"
                        field={this.props.field.key}
                        id={this.props.field.key}
                        placeholder={this.props.field.placeholder}
                    />
                );
            case 'password':
                return (
                    <Text
                        type="password"
                        className="border rounded w-full p-3 border-base-300"
                        field={this.props.field.key}
                        id={this.props.field.key}
                        placeholder={this.props.field.placeholder}
                    />
                );
            case 'select':
                return (
                    <Select
                        field={this.props.field.key}
                        id={this.props.field.key}
                        options={this.props.field.options}
                        placeholder={this.props.field.placeholder}
                        className="border rounded w-full p-3 border-base-300"
                    />
                );
            case 'multiselect':
                return (
                    <MultiSelect
                        key={this.props.field.key}
                        multi
                        onChange={handleMultiSelectChange()}
                        options={this.props.field.options}
                        placeholder={this.props.field.placeholder}
                        removeSelected
                        simpleValue
                        value={this.props.formApi.values[this.props.field.key]}
                        className="text-base-600 font-400 w-full"
                    />
                );
            default:
                return '';
        }
    }
}

export default FormField;
