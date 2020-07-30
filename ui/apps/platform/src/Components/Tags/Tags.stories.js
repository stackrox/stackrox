/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import Tags from 'Components/Tags';

export default {
    title: 'Tags',
    component: Tags,
};

const defaultTags = ['spicy', 'mild', 'bland'];

export const withCollapsibleTitle = () => {
    const [tags, setTags] = useState(defaultTags);

    function onChange(data) {
        setTags(data);
    }

    return <Tags title="Spice Levels" tags={tags} onChange={onChange} defaultOpen />;
};

export const withCollapsibleTitleClosed = () => {
    const [tags, setTags] = useState(defaultTags);

    function onChange(data) {
        setTags(data);
    }

    return <Tags title="Spice Levels" tags={tags} onChange={onChange} defaultOpen={false} />;
};

export const withNonCollapsibleTitle = () => {
    const [tags, setTags] = useState(defaultTags);

    function onChange(data) {
        setTags(data);
    }

    return (
        <Tags
            title="Spice Levels"
            tags={tags}
            onChange={onChange}
            defaultOpen
            isCollapsible={false}
        />
    );
};

export const withNoTitle = () => {
    const [tags, setTags] = useState(defaultTags);

    function onChange(data) {
        setTags(data);
    }

    return <Tags tags={tags} onChange={onChange} defaultOpen />;
};

export const withDisabledInput = () => {
    const [tags, setTags] = useState(defaultTags);

    function onChange(data) {
        setTags(data);
    }

    return <Tags tags={tags} onChange={onChange} defaultOpen isDisabled />;
};

export const withLoadingInput = () => {
    const [tags, setTags] = useState(defaultTags);

    function onChange(data) {
        setTags(data);
    }

    return <Tags tags={tags} onChange={onChange} defaultOpen isLoading />;
};
