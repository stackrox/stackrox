import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { FormField } from 'react-form';

class NumericInput extends Component {
    static propTypes = {
        fieldApi: PropTypes.shape({
            setValue: PropTypes.func,
            getValue: PropTypes.func,
            setTouched: PropTypes.func
        }).isRequired,
        onChange: PropTypes.func,
        onBlur: PropTypes.func
    }

    static defaultProps = {
        onChange: null,
        onBlur: null
    }

    onChange = (e) => {
        const { fieldApi, onChange } = this.props;
        const value = parseInt(e.target.value, 10);
        fieldApi.setValue(value);
        if (onChange) {
            onChange(e.target.value, e);
        }
    }

    onBlur = (e) => {
        const { fieldApi, onBlur } = this.props;
        fieldApi.setTouched();
        if (onBlur) {
            onBlur(e);
        }
    }

    render() {
        const { fieldApi, ...rest } = this.props;
        return (
            <input
                {...rest}
                type="number"
                value={fieldApi.getValue() || ''}
                onChange={this.onChange}
                onBlur={this.onBlur}
            />
        );
    }
}

export default FormField(NumericInput);
