import React from 'react';
import PropTypes from 'prop-types';

import * as Icon from 'react-feather';

const iconClassName = 'h-4 w-4 mb-1';

export const navLinks = [
    {
        text: 'Dashboard',
        to: '/main/dashboard',
        renderIcon: () => <Icon.BarChart2 className={iconClassName} />
    },
    {
        text: 'Network',
        to: '/main/network',
        renderIcon: () => <Icon.Share2 className={iconClassName} />
    },
    {
        text: 'Violations',
        to: '/main/violations',
        renderIcon: () => <Icon.AlertTriangle className={iconClassName} />
    },
    {
        text: 'Compliance',
        to: '/main/compliance',
        renderIcon: () => <Icon.CheckSquare className={iconClassName} />
    },
    {
        text: 'Config Management',
        to: '/main/configmanagement',
        renderIcon: () => <Icon.CheckSquare className={iconClassName} />,
        devOnly: true
    },
    {
        text: 'Risk',
        to: '/main/risk',
        renderIcon: () => <Icon.ShieldOff className={iconClassName} />
    },
    {
        text: 'Images',
        to: '/main/images',
        renderIcon: () => <Icon.FileMinus className={iconClassName} />
    },
    {
        text: 'Secrets',
        to: '/main/secrets',
        renderIcon: () => <Icon.Lock className={iconClassName} />
    },
    {
        text: 'Configure',
        to: '',
        renderIcon: () => <Icon.Settings className={iconClassName} />,
        panelType: 'configure'
    }
];

const filteredNavLinks = navLinks.filter(navLink =>
    process.env.NODE_ENV === 'development' ? true : !navLink.devOnly
);

const LeftSideNavLinks = ({ renderLink }) => (
    <ul className="flex flex-col list-reset uppercase text-sm tracking-wide">
        {filteredNavLinks.map(navLink => (
            <li key={navLink.text}>{renderLink(navLink)}</li>
        ))}
    </ul>
);

LeftSideNavLinks.propTypes = {
    renderLink: PropTypes.func.isRequired
};

export default LeftSideNavLinks;
