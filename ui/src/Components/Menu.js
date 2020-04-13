import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const optionsClass =
    'flex items-center relative text-left px-2 py-3 text-sm border-b border-base-400 hover:bg-base-200 capitalize';

const Menu = ({
    buttonClass,
    buttonContent,
    menuClassName,
    className,
    options,
    disabled,
    grouped
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
    const onClickHandler = () => () => {
        if (!isMenuOpen) showMenu();
        else hideMenu();
    };

    function renderOptions(formattedOptions) {
        // TO DO: use accessibility friendly semantic HTML elements (<li>, <ul>)
        return formattedOptions.map(
            ({ className: optionClassName, link, label, icon, onClick }) => {
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
        return Object.keys(formattedOptions).map(group => {
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

    return (
        <div className={`${className} inline-block relative z-50`}>
            <button
                className={`flex h-full w-full ${buttonClass}`}
                type="button"
                onClick={onClickHandler()}
                disabled={disabled}
                data-testid="menu-button"
            >
                {buttonContent}
            </button>
            {isMenuOpen && (
                <div
                    className={`absolute bg-white flex flex-col flex-no-wrap menu right-0 z-50 min-w-32 bg-base-100 shadow ${menuClassName}`}
                    data-testid="menu-list"
                >
                    {grouped ? renderGroupedOptions(options) : renderOptions(options)}
                </div>
            )}
        </div>
    );
};

Menu.propTypes = {
    buttonClass: PropTypes.string,
    buttonContent: PropTypes.node.isRequired,
    menuClassName: PropTypes.string,
    className: PropTypes.string,
    options: PropTypes.oneOfType([
        PropTypes.arrayOf(
            PropTypes.shape({
                className: PropTypes.string,
                icon: PropTypes.object,
                label: PropTypes.string.isRequired,
                link: PropTypes.string,
                onClick: PropTypes.func
            })
        ).isRequired,
        PropTypes.shape({})
    ]).isRequired,
    disabled: PropTypes.bool,
    grouped: PropTypes.bool
};

Menu.defaultProps = {
    buttonClass: '',
    disabled: false,
    menuClassName: '',
    className: '',
    grouped: false
};

export default Menu;
