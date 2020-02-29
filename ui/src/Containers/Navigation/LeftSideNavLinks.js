import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { connect } from 'react-redux';

import { knownBackendFlags } from 'utils/featureFlags';
import { filterLinksByFeatureFlag } from './navHelpers';

const iconClassName = 'h-4 w-4';

export const navLinks = [
    {
        text: 'Dashboard',
        to: '/main/dashboard',
        renderIcon: () => <Icon.BarChart2 className={iconClassName} />
    },
    {
        text: 'Network Graph',
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
        text: 'Vulnerability Management',
        to: '/main/vulnerability-management',
        renderIcon: () => <Icon.Layers className={iconClassName} />,
        featureFlag: knownBackendFlags.ROX_VULN_MGMT_UI
    },
    {
        text: 'Configuration Management',
        to: '/main/configmanagement',
        renderIcon: () => <Icon.UserCheck className={iconClassName} />
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
        text: 'Platform Configuration',
        to: '',
        renderIcon: () => <Icon.Settings className={iconClassName} />,
        panelType: 'configure',
        data: 'configure'
    }
];

const LeftSideNavLinks = ({ renderLink, featureFlags }) => (
    <ul className="flex flex-col list-reset uppercase text-sm tracking-wide">
        {filterLinksByFeatureFlag(featureFlags, navLinks).map(navLink => (
            <li key={navLink.text}>{renderLink(navLink)}</li>
        ))}
    </ul>
);

LeftSideNavLinks.propTypes = {
    renderLink: PropTypes.func.isRequired,
    featureFlags: PropTypes.arrayOf(
        PropTypes.shape({
            envVar: PropTypes.string.isRequired,
            enabled: PropTypes.bool.isRequired
        })
    ).isRequired
};

const mapStateToProps = createStructuredSelector({
    featureFlags: selectors.getFeatureFlags
});

export default connect(mapStateToProps)(LeftSideNavLinks);
