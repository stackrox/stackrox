import React, { useState } from 'react';
import * as Icon from 'react-feather';
import PropTypes from 'prop-types';
import { Manager, Target, Popper, Arrow } from 'react-popper';
import onClickOutside from 'react-onclickoutside';

import { useTheme } from 'Containers/ThemeProvider';
import { getDarkModeLinkClassName } from './navHelpers';

const modifiers = {
    customStyle: {
        enabled: true,
        fn: data => {
            Object.assign(data.styles, {
                left: '133px' // Left navigation width
            });
            return data;
        }
    }
};

const menuLinkClassName =
    'block p-4 border-b border-base-400 no-underline text-primary-800 hover:text-base-700 hover:bg-base-200';

const ApiDocsMenu = () => (
    <ul
        data-test-id="api-docs-menu"
        className="uppercase bg-base-100 border-2 border-base-400 shadow-lg rounded text-center text-base-100"
    >
        <li>
            <a
                href="/docs/product"
                target="_blank"
                rel="noopener noreferrer"
                className={menuLinkClassName}
            >
                Documentation
            </a>
        </li>
        <li>
            <a
                href="/main/apidocs"
                target="_blank"
                rel="noopener noreferrer"
                className={menuLinkClassName}
            >
                API Reference
            </a>
        </li>
    </ul>
);

const ApiDocsNavigation = ({ onClick }) => {
    const [toggleMenu, setToggleMenu] = useState(false);
    const { isDarkMode } = useTheme();

    const linkClassName = `${getDarkModeLinkClassName(
        isDarkMode
    )} w-full font-condensed font-700 text-primary-400 flex px-3 no-underline h-18 items-center border-b`;

    ApiDocsNavigation.handleClickOutside = () => {
        setToggleMenu(false);
    };

    function onButtonClick() {
        setToggleMenu(!toggleMenu);
        if (onClick) onClick();
    }

    return (
        <Manager>
            <Target>
                <button
                    type="button"
                    data-test-id="api-docs"
                    className={`${linkClassName} border-t`}
                    onClick={onButtonClick}
                >
                    <div className="text-center pt-1 pr-2">
                        <Icon.HelpCircle className="h-4 w-4 text-primary-400" />
                    </div>
                    <div
                        className={`text-center ${
                            isDarkMode ? 'text-base-600' : 'text-base-100'
                        } font-condensed uppercase text-sm tracking-wide`}
                    >
                        Help Center
                    </div>
                </button>
            </Target>
            <Popper
                className={`popper ${toggleMenu ? '' : 'hidden'} ${
                    isDarkMode ? 'theme-dark' : 'theme-light'
                }`}
                placement="right"
                modifiers={modifiers}
            >
                <Arrow className="arrow-left absolute" />
                <ApiDocsMenu />
            </Popper>
        </Manager>
    );
};

ApiDocsNavigation.propTypes = {
    onClick: PropTypes.func
};

ApiDocsNavigation.defaultProps = {
    onClick: null
};

const clickOutsideConfig = {
    handleClickOutside: () => ApiDocsNavigation.handleClickOutside
};

export default onClickOutside(ApiDocsNavigation, clickOutsideConfig);
