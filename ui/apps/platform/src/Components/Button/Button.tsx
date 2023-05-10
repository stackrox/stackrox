import React from 'react';
import { ClipLoader } from 'react-spinners';

const Loader = ({ size }: { size: number }) => (
    <ClipLoader loading size={size} color="currentColor" />
);

export type ButtonProps = {
    dataTestId?: string | null;
    className?: string;
    icon?: React.ReactElement | null;
    text?: string | null;
    textCondensed?: string | null;
    textClass?: string;
    onClick?: () => void;
    disabled?: boolean;
    isLoading?: boolean;
    loaderSize?: number;
    tabIndex?: number;
};

const noopOnClick = () => {};

const Button = ({
    dataTestId = null,
    className = '',
    icon = null,
    text = null,
    textCondensed = null,
    textClass = '',
    onClick = noopOnClick,
    disabled = false,
    isLoading = false,
    loaderSize = 20,
    tabIndex,
    ...ariaProps
}: ButtonProps) => {
    const content = (
        <div className="flex items-center">
            {icon}
            {textCondensed ? (
                <>
                    <span className={`${textClass} lg:hidden`}> {textCondensed} </span>
                    <span className="hidden lg:block"> {text} </span>
                </>
            ) : (
                text
            )}
        </div>
    );
    return (
        <button
            type="button"
            className={className}
            onClick={onClick}
            disabled={disabled}
            data-testid={dataTestId}
            tabIndex={tabIndex}
            {...ariaProps}
        >
            {isLoading ? <Loader size={loaderSize} /> : content}
        </button>
    );
};

export default Button;
