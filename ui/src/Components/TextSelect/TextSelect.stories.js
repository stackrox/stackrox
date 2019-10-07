import React from 'react';
import TextSelect from './TextSelect';

export default {
    title: 'TextSelect',
    component: TextSelect
};

export const withOptions = () => {
    const options = [
        { label: 'Apple', value: 'Apple' },
        { label: 'Banana', value: 'Banana' },
        { label: 'Orange', value: 'Orange' }
    ];
    const { value } = options[0];
    function onChange() {}
    return <TextSelect value={value} options={options} onChange={onChange} />;
};
