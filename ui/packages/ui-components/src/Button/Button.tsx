import React, { ReactElement, ReactNode } from 'react';

export type HOCButtonProps = {
    type?: 'button' | 'submit';
    onClick?: React.MouseEventHandler<HTMLButtonElement>; // required for type "button", but not for type "submit"
    children: ReactNode;
};

export type ButtonProps = {
    type?: 'button' | 'submit';
    onClick?: React.MouseEventHandler<HTMLButtonElement>; // required for type "button", but not for type "submit"
    children: ReactNode;
    colorType?: 'alert' | 'success' | 'base';
    isCondensed?: boolean;
};

const baseButtonClassName =
    'border-2 font-600 inline-flex items-center justify-center rounded-sm uppercase text-sm';
const baseClassName =
    'border-base-400 bg-base-100 hover:bg-base-200 hover:text-base-700 text-base-800';
const alertClassName =
    'border-alert-400 bg-alert-100 hover:bg-alert-200 hover:text-alert-700 text-alert-800';
const successClassName =
    'border-success-500 bg-success-200 hover:bg-success-300 hover:text-success-800 text-success-700';

function getColorClassName(colorType: ButtonProps['colorType']): string {
    switch (colorType) {
        case 'alert':
            return alertClassName;
        case 'success':
            return successClassName;
        case 'base':
        default:
            return baseClassName;
    }
}

function getPaddingClassName(isCondensed: boolean): string {
    return isCondensed ? 'px-2' : 'p-2';
}

// @TODO This is just starter code for the Button Component. We can discuss, in more detail, how we want to go about it later
/* Maybe omit type prop and separate into 2 components:
 * Button has onClick and children props
 * SubmitButton has children prop
 */
function Button({
    type = 'button',
    onClick,
    children,
    colorType = 'base',
    isCondensed = false,
}: ButtonProps): ReactElement {
    const colorClassName = getColorClassName(colorType);
    const paddingClassName = getPaddingClassName(isCondensed);
    const className = `${baseButtonClassName} ${colorClassName} ${paddingClassName}`;

    if (type === 'submit') {
        return (
            <button className={className} type="submit">
                {children}
            </button>
        );
    }

    return (
        <button className={className} type="button" onClick={onClick}>
            {children}
        </button>
    );
}

export default Button;
