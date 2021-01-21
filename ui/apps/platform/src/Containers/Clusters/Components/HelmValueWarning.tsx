import React, { ReactElement } from 'react';
import { inputTextClassName } from '../cluster.helpers';

export type HelmValueWarningProps = {
    currentValue: unknown;
    helmValue: unknown;
};

function HelmValueWarning({ currentValue, helmValue }: HelmValueWarningProps): ReactElement | null {
    // Note: it is not the recommended or performant to let a conponent decide for itself whether to render or not
    //       However, in this case, conditional rendering with long dereference change in the parent form were less readable,
    //       and the number of these components on the form is finite and small, the performance hit is negligible.
    if (helmValue === undefined || currentValue === helmValue) {
        return null;
    }

    let normalizedValue = '-';
    switch (typeof helmValue) {
        case 'boolean': {
            normalizedValue = helmValue ? 'true' : 'false';
            break;
        }
        case 'string': {
            normalizedValue = helmValue === '' ? '(empty string)' : helmValue;
            break;
        }
        default: {
            normalizedValue = String(helmValue);
        }
    }
    return (
        <div className={`${inputTextClassName} border-warning-300 bg-warning-300`}>
            Value in current Helm chart is: <code>{normalizedValue}</code>
        </div>
    );
}

export default HelmValueWarning;
