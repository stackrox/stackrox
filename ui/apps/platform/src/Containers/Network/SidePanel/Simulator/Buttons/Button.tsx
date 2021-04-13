import React, { ReactElement } from 'react';

type ButtonProps = {
    dataTestId: string;
    icon: ReactElement;
    text: string;
    onClick: () => void;
    disabled: boolean;
};

function Button({
    dataTestId,
    icon = <></>,
    text,
    onClick,
    disabled = false,
}: ButtonProps): ReactElement {
    return (
        <button
            type="button"
            className="inline-block flex items-center my-3 px-3 text-center bg-primary-600 font-700 rounded-sm text-base-100 h-9 hover:bg-primary-700"
            onClick={onClick}
            disabled={disabled}
            data-testid={dataTestId}
        >
            {icon}
            {text}
        </button>
    );
}

export default Button;
