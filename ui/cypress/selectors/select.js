/**
 * Selectors to interact with react-select
 */

const multiSelect = {
    input: '.react-select__input > input',
    values: '.react-select__multi-value__label',
    removeValueButton: (value) =>
        `.react-select__multi-value__label:contains("${value}") + .react-select__multi-value__remove`,
    options: '.react-select__option',
};

const singleSelect = {
    input: '.react-select__control',
    value: '.react-select__single-value',
    options: '.react-select__option',
};

const selectors = {
    multiSelect,
    singleSelect,
};

export default selectors;
