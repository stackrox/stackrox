/**
 * Selectors to interact with react-select
 */

const multiSelect = {
    dropdown: '.react-select__dropdown-indicator',
    input: '.react-select__input > input',
    values: '.react-select__multi-value__label',
    removeValueButton: (value) =>
        `.react-select__multi-value__label:contains("${value}") + .react-select__multi-value__remove`,
    options: '.react-select__option',
    placeholder: '.react-select__placeholder',
};

const singleSelect = {
    input: '.react-select__control input',
    value: '.react-select__single-value',
    options: '.react-select__option',
    menu: '.select__menu',
};

const selectors = {
    multiSelect,
    singleSelect,
};

export default selectors;
