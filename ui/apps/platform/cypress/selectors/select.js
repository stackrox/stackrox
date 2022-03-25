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
    input: '.react-select__control',
    value: '.react-select__single-value',
    options: '.react-select__option',
};

const patternFlySelect = {
    openMenu: '.pf-c-select__menu',
};

const selectors = {
    multiSelect,
    singleSelect,
    patternFlySelect,
};

export default selectors;
