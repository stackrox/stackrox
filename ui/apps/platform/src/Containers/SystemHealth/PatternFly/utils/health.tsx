import React, { ElementType, ReactElement } from 'react';

export const nbsp = '\u00A0'; // no-break space

export type CategoryStyle = {
    Icon: ElementType;
    bgColor: string;
    fgColor: string;
};

// Placeholder for absence of an icon because count of healthy and unhealthy entities are zero.
const Icon0 = ({ className }: { className: string }): ReactElement => <div className={className} />;

export const style0: CategoryStyle = {
    Icon: Icon0,
    bgColor: 'bg-base-200',
    fgColor: 'text-base-500',
};

export const style0PF: CategoryStyle = {
    Icon: Icon0,
    bgColor: 'pf-u-background-color-200',
    fgColor: 'pf-u-color-400',
};

export type CountMap = Record<string, number>;
export type LabelMap = Record<string, string>;
export type StyleMap = Record<string, CategoryStyle>;

/*
 * Return style of the most severe problem,
 * assuming that problem keys are in increasing order of severity.
 */
export const getProblemStyle = (
    countMap: CountMap,
    healthyKey: string,
    styleMap: StyleMap
): CategoryStyle => {
    let style = style0;

    Object.keys(countMap).forEach((key) => {
        if (key !== healthyKey && countMap[key] !== 0) {
            if (styleMap[key]) {
                style = styleMap[key];
            }
        }
    });

    return style;
};

export type CountableText = {
    plural: string;
    singular: string;
};

/*
 * The caller is responsible to concatenate count and text, if necessary.
 */
export const getCountableText = ({ plural, singular }: CountableText, count: number): string =>
    count === 1 ? singular : plural;
