/* eslint-disable no-nested-ternary */
import React from 'react';
import { Link } from 'react-router-dom';

type TabNavProps = {
    tabLinks: {
        title: string;
        href: string;
    }[];
    currentTabTitle?: string;
    isDisabled?: boolean;
};

function TabNav({ tabLinks, currentTabTitle, isDisabled }: TabNavProps) {
    return (
        <nav className="pf-c-nav pf-m-tertiary">
            <ul className="pf-c-nav__list">
                {tabLinks.map(({ title, href }) => {
                    const isCurrent = currentTabTitle === title;
                    const className = isCurrent ? 'pf-c-nav__link pf-m-current' : 'pf-c-nav__link';

                    return (
                        <li key={title} className="pf-c-nav__item">
                            {isDisabled ? (
                                <span className={className}>{title}</span>
                            ) : isCurrent ? (
                                <Link to={href} className={className} aria-current="page">
                                    {title}
                                </Link>
                            ) : (
                                <Link to={href} className={className}>
                                    {title}
                                </Link>
                            )}
                        </li>
                    );
                })}
            </ul>
        </nav>
    );
}

export default TabNav;
