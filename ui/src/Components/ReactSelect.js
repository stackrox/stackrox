import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactSelect, { components as selectComponents } from 'react-select';
import ReactSelectCreatable from 'react-select/lib/Creatable';

const defaultClassName = 'text-base-600 font-400 w-full';

const defaultComponentClassNames = {
    multiValue: 'bg-primary-200 border border-primary-300 text-primary-700'
};

const Control = ({ className, ...props }) => (
    <selectComponents.Control
        {...props}
        className={`${className} cursor-text border-2 border-base-300 flex items-center items-center text-xl shadow-none overflow-auto hover:border-base-400`}
    />
);

const Menu = ({ className, ...props }) => (
    <selectComponents.Menu className={`${className} z-10`} {...props} />
);

const MultiValue = props => (
    <selectComponents.MultiValue {...props} className={defaultComponentClassNames.multiValue} />
);

const defaultComponents = { Control, Menu, MultiValue };

/**
 * Adds the following changes to the react-select component:
 *   1. Changes the default styling through the default value for className property
 *   2. onChange callback receives (option value(s), option(s), ...remaining params from react-select)
 *   3. value property expects only option value (not the whole option object with label and value)
 */
function withAdjustedBehavior(SelectComponent) {
    return class extends Component {
        static propTypes = {
            /* Note: getOptionValue isn't fully supported by react-select Creatable component, it's recommended to use { label, value } options */
            getOptionValue: PropTypes.func,
            /* See react-select docs */
            className: PropTypes.string,
            /* See react-select docs */
            classNamePrefix: PropTypes.string,
            /* Callback for change events on the select invoked with params (selectedValues, selectedOptions, changeAction) */
            onChange: PropTypes.func,
            /* Passed components will be merged with the default components allowing passed components to override default ones */
            components: PropTypes.shape({}), // see react-select docs for the exact shape
            /* Renamed react-select 'value' property that accepts full option objects for the select value */
            optionValue: PropTypes.oneOfType([PropTypes.object, PropTypes.array]),
            /* The value of the select reflected by the values of the selected option(s), ignored if optionValue is passed */
            value: PropTypes.any, // eslint-disable-line react/forbid-prop-types
            /* See react-select docs */
            options: PropTypes.arrayOf(PropTypes.object)
        };

        static defaultProps = {
            getOptionValue: option => option.value,
            className: defaultClassName,
            classNamePrefix: 'react-select', // only for testing purposes, not for CSS creation
            onChange: () => {},
            components: defaultComponents,
            optionValue: null,
            value: null,
            options: []
        };

        state = {
            createdOptions: []
        };

        // we have to keep the list of created options to be able to reference them by option value
        updateCreatedOptions = (onChangeValue, changeAction) => {
            const newValueArray = Array.isArray(onChangeValue) ? onChangeValue : [onChangeValue];
            const lastOption = newValueArray.length && onChangeValue[onChangeValue.length - 1];
            switch (changeAction.action) {
                case 'create-option':
                    this.setState(({ createdOptions }) => ({
                        createdOptions: [...createdOptions, lastOption]
                    }));
                    break;
                case 'remove-value':
                    this.setState(({ createdOptions }) => ({
                        createdOptions: createdOptions.filter(
                            option => option !== changeAction.removedValue
                        )
                    }));
                    break;
                default:
                // do nothing
            }
        };

        // we want to pass to the callback from props only value(s) as the first parameter
        onChange = (newValue, changeAction, ...rest) => {
            this.updateCreatedOptions(newValue, changeAction);
            const { getOptionValue, onChange } = this.props;
            const onlyValues =
                newValue && Array.isArray(newValue)
                    ? newValue.map(option => getOptionValue(option))
                    : getOptionValue(newValue);
            onChange(onlyValues, newValue, changeAction, ...rest);
        };

        transformValue = (getOptionValue, options, value, optionValue) => {
            if (optionValue) return optionValue;

            const allOptions = options.concat(this.state.createdOptions);
            return Array.isArray(value)
                ? allOptions.filter(option => value.includes(getOptionValue(option)))
                : allOptions.find(option => getOptionValue(option) === value);
        };

        render() {
            const {
                getOptionValue,
                onChange,
                components,
                value,
                optionValue,
                options,
                ...rest
            } = this.props;
            const valueToPass = this.transformValue(getOptionValue, options, value, optionValue);
            const mergedComponents = {
                ...defaultComponents,
                ...components
            };

            return (
                <SelectComponent
                    getOptionValue={getOptionValue}
                    onChange={this.onChange}
                    components={mergedComponents}
                    value={valueToPass}
                    options={options}
                    {...rest}
                />
            );
        }
    };
}

export default withAdjustedBehavior(ReactSelect);

export const Creatable = withAdjustedBehavior(ReactSelectCreatable);
