import React, { ReactElement } from 'react';
import { IconProps } from 'react-feather';

export type ToggleButtonProps = {
    icon?: React.FC<IconProps>;
    value: string;
    text: string;
    onClick: (value: string) => void;
    isActive?: boolean;
};

const buttonClassName =
    'first:rounded-l last:rounded-r bg-base-100 border-2 border-base-400 font-600 inline-flex items-center justify-center m-0 py-1 px-2 text-sm uppercase';
const inactiveButtonClassName = `${buttonClassName} first:border-r-0 last:border-l-0 hover:bg-base-200 hover:text-base-700 text-base-600`;
const activeButtonClassName = `${buttonClassName} text-primary-800 bg-primary-300 border-primary-600`;

function ToggleButton({ isActive, icon, value, text, onClick }: ToggleButtonProps): ReactElement {
    const className = isActive ? activeButtonClassName : inactiveButtonClassName;
    const Icon = icon;

    function onClickHandler(event: React.MouseEvent<HTMLButtonElement>): void {
        if (!isActive) {
            const element = event.target as HTMLButtonElement;
            onClick(element.value);
        }
    }

    return (
        <button
            className={className}
            type="button"
            onClick={onClickHandler}
            value={value}
            data-testid={isActive ? 'active-toggle-button' : 'toggle-button'}
        >
            {Icon && <Icon className="h-3 w-3 mr-1" />}
            {text}
        </button>
    );
}

export default ToggleButton;
