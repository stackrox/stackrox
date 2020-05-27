import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { ChevronDown } from 'react-feather';

import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';

const optionsClass =
    'flex items-center relative text-left px-2 py-3 text-sm border-b border-base-400 hover:bg-base-200 capitalize';

const Menu = ({
    buttonClass,
    buttonText,
    buttonIcon,
    menuClassName,
    className,
    options,
    disabled,
    grouped,
    tooltip,
    dataTestId,
    hideCaret,
    buttonTextClassName,
}) => {
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
        if (!isMenuOpen) showMenu();
        else hideMenu();
    };

    function renderOptions(formattedOptions) {
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

    function renderGroupedOptions(formattedOptions) {
        return Object.keys(formattedOptions).map((group) => {
            return (
                <React.Fragment key={group}>
                    <div className="uppercase font-condensed p-3 border-b border-primary-300 text-lg">
                        {group}
                    </div>
                    <div className="px-2">{renderOptions(options[group])}</div>
                </React.Fragment>
            );
        });
    }

    const tooltipClassName = !tooltip || disabled ? 'invisible' : '';
    return (
        <Tooltip content={<TooltipOverlay>{tooltip}</TooltipOverlay>} className={tooltipClassName}>
            <div className={`${className} inline-block relative z-10`}>
                <button
                    className={`flex h-full w-full ${buttonClass}`}
                    type="button"
                    onClick={onClickHandler()}
                    disabled={disabled}
                    data-testid={dataTestId}
                >
                    <div className="flex flex-1 justify-center items-center text-left">
                        {buttonIcon}
                        {buttonText && <span className={buttonTextClassName}>{buttonText}</span>}
                        {!hideCaret && <ChevronDown className="h-4 ml-1 pointer-events-none w-4" />}
                    </div>
                </button>
                {isMenuOpen && (
                    <div
                        className={`absolute flex flex-col flex-no-wrap menu right-0 z-10 min-w-32 bg-base-100 shadow border border-base-400 ${menuClassName}`}
                        data-testid="menu-list"
                    >
                        {grouped ? renderGroupedOptions(options) : renderOptions(options)}
                    </div>
                )}
            </div>
        </Tooltip>
    );
};

Menu.propTypes = {
    buttonClass: PropTypes.string,
    buttonText: PropTypes.string.isRequired,
    buttonTextClassName: PropTypes.string,
    buttonIcon: PropTypes.node,
    menuClassName: PropTypes.string,
    className: PropTypes.string,
    options: PropTypes.oneOfType([
        PropTypes.arrayOf(
            PropTypes.shape({
                className: PropTypes.string,
                icon: PropTypes.object,
                label: PropTypes.string,
                link: PropTypes.string,
                onClick: PropTypes.func,
                component: PropTypes.node,
            })
        ).isRequired,
        PropTypes.shape({}),
    ]).isRequired,
    disabled: PropTypes.bool,
    grouped: PropTypes.bool,
    tooltip: PropTypes.string,
    dataTestId: PropTypes.string,
    hideCaret: PropTypes.bool,
};

Menu.defaultProps = {
    buttonClass: '',
    buttonTextClassName: '',
    buttonIcon: null,
    disabled: false,
    menuClassName: '',
    className: '',
    grouped: false,
    tooltip: '',
    dataTestId: 'menu-button',
    hideCaret: false,
};

export default Menu;
