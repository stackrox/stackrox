import React from 'react';

import TextArea from 'Components/forms/TextArea';

export default {
    title: 'TextArea',
    component: TextArea
};

export const withText = () => {
    const value = 'Now, this is a story all about how my life got flipped - turned upside down';
    function register() {}
    const errors = {};
    return (
        <TextArea
            name="message"
            required
            register={register}
            errors={errors}
            rows="5"
            cols="33"
            defaultValue={value}
            placeholder="Write a comment here..."
        />
    );
};
