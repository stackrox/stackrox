import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const optionsClass =
    'flex items-center relative text-left px-2 py-3 border-b border-base-300 hover:bg-base-200';

const Menu = ({ buttonClass, buttonContent, className, options }) => {
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

    const renderOptions = options.map(option => {
        if (option.link) {
            return (
                <Link
                    to={option.link}
                    className={`${optionsClass} ${option.className} no-underline text-base-600`}
                    key={option.label}
                    data-test-id={option.label}
                >
                    {option.icon}
                    {option.label && <span className="pl-2">{option.label}</span>}
                </Link>
            );
        }

        return (
            <button
                type="button"
                className={`${optionsClass} ${option.className}`}
                onClick={option.onClick}
                key={option.label}
                data-test-id={option.label}
            >
                {option.icon}
                {option.label && <span className="pl-2">{option.label}</span>}
            </button>
        );
    });

    return (
        <div className={`${className} inline-block relative z-60`}>
            <button
                className={`flex h-full w-full ${buttonClass}`}
                type="button"
                onClick={onClickHandler()}
            >
                {buttonContent}
            </button>
            {isMenuOpen && (
                <div
                    className="absolute bg-white flex flex-col flex-no-wrap menu pin-r z-60 min-w-43 bg-base-100 shadow"
                    data-test-id="menu-list"
                >
                    {renderOptions}
                </div>
            )}
        </div>
    );
};

Menu.propTypes = {
    buttonClass: PropTypes.string,
    buttonContent: PropTypes.node.isRequired,
    className: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(
        PropTypes.shape({
            className: PropTypes.string,
            icon: PropTypes.func,
            label: PropTypes.string.isRequired,
            link: PropTypes.string,
            onClick: PropTypes.func
        })
    ).isRequired
};

Menu.defaultProps = {
    buttonClass: ''
};

export default Menu;
