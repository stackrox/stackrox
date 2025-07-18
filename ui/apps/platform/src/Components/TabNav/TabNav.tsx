import React from 'react';
import { Link } from 'react-router-dom-v5-compat';

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
        <nav className="pf-v5-c-nav pf-m-tertiary">
            <ul className="pf-v5-c-nav__list">
                {tabLinks.map(({ title, href }) => {
                    const isCurrent = currentTabTitle === title;
                    const className = isCurrent
                        ? 'pf-v5-c-nav__link pf-m-current'
                        : 'pf-v5-c-nav__link';

                    return (
                        <li key={title} className="pf-v5-c-nav__item">
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
