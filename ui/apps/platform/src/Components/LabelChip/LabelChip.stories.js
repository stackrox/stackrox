import React from 'react';

import LabelChip from './LabelChip';

export default {
    title: 'LabelChip',
    component: LabelChip,
};

function doNothing() {}

export const withTypeAlert = () => <LabelChip type="alert" text="14 C" />;

export const withTypeCaution = () => <LabelChip type="caution" text="9 H" />;

export const withTypeWarning = () => <LabelChip type="warning" text="35 M" />;

export const withTypeBase = () => <LabelChip type="base" text="14 L" />;

export const withTypePrimary = () => <LabelChip type="primary" text="Thanks for reading me" />;

export const withTypeSecondary = () => (
    <LabelChip type="secondary" text="Environment Impact: 75%" />
);

export const withTypeTertiary = () => <LabelChip type="tertiary" text="13 Images" />;

export const withTypeAccent = () => <LabelChip type="accent" text="Bueños Dios Amigo" />;

export const withTypeSuccess = () => <LabelChip type="success" text="Fixable" />;

export const withOnClick = () => <LabelChip type="base" text="Link" onClick={doNothing} />;
