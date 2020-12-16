import React, { ReactElement, ReactNode } from 'react';
import { Info, Check, AlertTriangle } from 'react-feather';

export type MessageProps = {
    extraClasses?: string;
    extraBodyClasses?: string;
    children: ReactNode;
    icon?: ReactElement;
    type?: 'base' | 'success' | 'warn' | 'error';
};

export const baseClasses =
    'flex p-4 rounded items-stretch leading-normal flex-shrink-0 w-full border';

const wrapperVariants = {
    base: 'base-message bg-base-200 border-base-500 text-base-700',
    success: 'success-message bg-success-200 border-success-700 text-success-800',
    warn: 'warn-message bg-warning-200 border-warning-700 text-warning-800',
    error: 'error-message bg-alert-200 border-alert-700 text-alert-800',
};

const bodyVariants = {
    base: 'border-base-500',
    success: 'border-success-600',
    warn: 'border-warning-700',
    error: 'border-alert-700',
};

const iconVariants = {
    base: <Info className="h-6 w-6" strokeWidth="2px" data-testid="info-icon" />,
    success: <Check className="h-6 w-6" strokeWidth="2px" />,
    warn: <AlertTriangle className="h-6 w-6" strokeWidth="2px" />,
    error: <AlertTriangle className="h-6 w-6" strokeWidth="2px" />,
};

function Message({
    children,
    extraClasses = '',
    extraBodyClasses = '',
    icon,
    type = 'base',
}: MessageProps): ReactElement {
    const variantClasses = wrapperVariants[type] ?? '';
    const variantBodyClasses = bodyVariants[type] ?? '';
    const variantIcon = iconVariants[type];

    return (
        <div className={`${baseClasses} ${variantClasses} ${extraClasses}`} data-testid="message">
            <div className="flex items-center justify-start flex-shrink-0 pr-4">
                <div className="flex p-4 rounded-full shadow-lg bg-base-100">
                    {icon || variantIcon}
                </div>
            </div>
            <div
                className={`flex items-center pl-3 border-l ${variantBodyClasses} ${extraBodyClasses}`}
                data-testid="message-body"
            >
                <div>{children}</div>
            </div>
        </div>
    );
}

export default Message;
