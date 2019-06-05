import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const optionsClass = 'text-left p-4 border-b border-base-300 hover:bg-base-200';

const Menu = ({ triggerComponent, className, options }) => {
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
                    className={`${optionsClass} no-underline text-base-600`}
                    key={option.label}
                    data-test-id={option.label}
                >
                    {option.label}
                </Link>
            );
        }
        return (
            <button
                type="button"
                className={optionsClass}
                onClick={option.onClick}
                key={option.label}
                data-test-id={option.label}
            >
                {option.label}
            </button>
        );
    });

    return (
        <div className={`${className} inline-block relative z-60`}>
            <button className="flex h-full w-full" type="button" onClick={onClickHandler()}>
                {triggerComponent}
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
    triggerComponent: PropTypes.node.isRequired,
    className: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(
        PropTypes.shape({
            label: PropTypes.string.isRequired,
            link: PropTypes.string,
            onClick: PropTypes.func
        })
    ).isRequired
};

export default Menu;
