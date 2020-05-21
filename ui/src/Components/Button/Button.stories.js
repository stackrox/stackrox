import React from 'react';

import Button from './Button';

export default {
    title: 'Button',
    component: Button,
};

export const withText = () => <Button text="Make It So" />;

export const withTextCondensed = () => <Button text="Export" textCondensed="Export" />;

export const withClassName = () => <Button className="btn btn-base h-10" text="Export" />;

export const withTextClass = () => <Button text="Delete" textClass="text-alert-600" />;

export const withDisabled = () => <Button disabled text="Make It So" />;

export const withIsLoading = () => <Button isLoading text="Make It So" />;
