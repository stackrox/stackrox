/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import Tags from 'Components/Tags';

export default {
    title: 'Tags',
    component: Tags
};

const defaultTags = ['spicy', 'mild', 'bland'];

export const withData = () => {
    const [tags, setTags] = useState(defaultTags);

    function onChange(data) {
        setTags(data);
    }

    return <Tags type="Violation" tags={tags} onChange={onChange} defaultOpen />;
};
