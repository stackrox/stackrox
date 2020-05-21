/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import SectionHeaderInput from './SectionHeaderInput';

export default {
    title: 'SectionHeaderInput',
    component: SectionHeaderInput,
};

export const withHeader = () => {
    const [value, setValue] = useState('policy section 1');
    const inputProps = { value, onChange: (e) => setValue(e.target.value) };
    return <SectionHeaderInput input={inputProps} />;
};

export const withReadOnly = () => {
    const [value, setValue] = useState('policy section 1');
    const inputProps = { value, onChange: (e) => setValue(e.target.value) };
    return <SectionHeaderInput input={inputProps} readOnly />;
};
