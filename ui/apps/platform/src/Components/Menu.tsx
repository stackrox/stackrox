import React, { useState, ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { ChevronDown, ChevronUp } from 'react-feather';
import { Tooltip } from '@patternfly/react-core';

const optionsClass =
    'flex items-center relative text-left px-2 py-3 text-sm border-b border-base-400 hover:bg-base-200 capitalize';

export interface MenuOption {
    className: string;
    icon: ReactElement;
    label: string;
    link?: string;
    onClick?: React.MouseEventHandler<HTMLButtonElement>;
    component?: ReactElement;
}

type GroupedMenuOptions = Record<string, MenuOption[]>;

interface MenuProps {
    buttonClass?: string;
    buttonText?: string;
    buttonTextClassName?: string;
    buttonIcon?: ReactElement;
    menuClassName?: string;
    className?: string;
    options?: GroupedMenuOptions | MenuOption[];
    disabled?: boolean;
    // TODO the `grouped` prop should be deprecated in favor of type narrowing once all dependent files are moved to TypeScript
    grouped?: boolean;
    tooltip?: string;
    dataTestId?: string;
    hideCaret?: boolean;
    customMenuContent?: ReactElement;
}

const Menu = ({
    options,
    buttonClass = '',
    buttonText = '',
    buttonIcon,
    menuClassName = '',
    className = '',
    disabled = false,
    grouped = false,
    tooltip = '',
    dataTestId = 'menu-button',
    hideCaret = false,
    buttonTextClassName = '',
    customMenuContent,
}: MenuProps) => {
    const [isMenuOpen, setMenuState] = useState(false);

    const hideMenu = () => {
        setMenuState(false);
        document.removeEventListener('click', hideMenu);
    };
    const showMenu = () => {
        setMenuState(true);
        document.addEventListener('click', hideMenu);
    };
    const onClickHandler = () => (e) => {
        e.stopPropagation();
        if (!isMenuOpen) {
            showMenu();
        } else {
            hideMenu();
        }
    };

    function renderOptions(formattedOptions: MenuOption[]) {
        // TO DO: use accessibility friendly semantic HTML elements (<li>, <ul>)
        return formattedOptions.map(
            ({ className: optionClassName, link, label, component, icon, onClick }) => {
                if (link) {
                    return (
                        <Link
                            to={link}
                            className={`${optionsClass} ${optionClassName} no-underline text-base-600`}
                            key={label}
                            data-testid={label}
                        >
                            {icon}
                            {label && <span className="pl-2">{label}</span>}
                            {component}
                        </Link>
                    );
                }

                return (
                    <button
                        type="button"
                        className={`${optionsClass} ${optionClassName}`}
                        onClick={onClick}
                        key={label}
                        data-testid={label}
                    >
                        {icon}
                        {label && <span className="pl-2">{label}</span>}
                    </button>
                );
            }
        );
    }

    function renderGroupedOptions(formattedOptions: GroupedMenuOptions) {
        return Object.keys(formattedOptions).map((group) => {
            return options ? (
                <React.Fragment key={group}>
                    <div className="p-3 border-b border-primary-300">{group}</div>
                    <div className="px-2">{renderOptions(options[group])}</div>
                </React.Fragment>
            ) : (
                []
            );
        });
    }

    const tooltipClassName = !tooltip || disabled ? 'invisible' : '';
    return (
        <Tooltip content={tooltip} className={tooltipClassName}>
            <div className={`${className} inline-block relative z-10`}>
                <button
                    className={`flex h-full w-full ${buttonClass}`}
                    type="button"
                    onClick={onClickHandler()}
                    disabled={disabled}
                    data-testid={dataTestId}
                >
                    <div className="flex flex-1 justify-center items-center text-left h-full">
                        {buttonIcon}
                        {buttonText && <span className={buttonTextClassName}>{buttonText}</span>}
                        {!hideCaret &&
                            (isMenuOpen ? (
                                <ChevronUp className="h-4 ml-1 pointer-events-none w-4" />
                            ) : (
                                <ChevronDown className="h-4 ml-1 pointer-events-none w-4" />
                            ))}
                    </div>
                </button>
                {isMenuOpen &&
                    // if `customMenuContent is provided, show it; otherwise, loop over the options
                    (customMenuContent || (
                        <div
                            className={`absolute flex flex-col flex-nowrap menu right-0 z-10 min-w-32 bg-base-100 shadow border border-base-400 ${menuClassName}`}
                            data-testid="menu-list"
                        >
                            {grouped
                                ? renderGroupedOptions(options as GroupedMenuOptions)
                                : renderOptions(options as MenuOption[])}
                        </div>
                    ))}
            </div>
        </Tooltip>
    );
};

export default Menu;
