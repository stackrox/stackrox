import React, { ReactElement, ReactNode } from 'react';

export type CheckboxWithLabelProps = {
    id: string;
    ariaLabel: string;
    checked: boolean;
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
    children: ReactNode;
    isDisabled?: boolean;
};

function CheckboxWithLabel({
    id,
    ariaLabel,
    checked,
    onChange,
    children,
    isDisabled = false,
}: CheckboxWithLabelProps): ReactElement {
    return (
        <div className="flex justify-center items-center">
            <input
                className="form-checkbox h-4 w-4 border-base-500 text-primary-500"
                type="checkbox"
                id={id}
                checked={!!checked}
                onChange={onChange}
                aria-label={ariaLabel}
                disabled={isDisabled}
            />
            <label htmlFor={id} className="pl-2">
                {children}
            </label>
        </div>
    );
}

export default CheckboxWithLabel;
