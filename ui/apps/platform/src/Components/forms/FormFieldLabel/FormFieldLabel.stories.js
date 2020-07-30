import React from 'react';

import FormFieldLabel from './FormFieldLabel';

export default {
    title: 'FormFieldLabel',
    component: FormFieldLabel,
};

export const withTextLabel = () => {
    return <FormFieldLabel text="Your Field" />;
};

export const withTextLabelAndRequiredMarker = () => {
    return <FormFieldLabel text="Your Field" required />;
};

export const withTextLabelAndRequiredMarkerIsEmpty = () => {
    return <FormFieldLabel text="Your Field" required empty />;
};
