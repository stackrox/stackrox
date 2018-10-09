import React, { Component } from 'react';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { withRouter, NavLink as Link } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import { createStructuredSelector } from 'reselect';
import find from 'lodash/find';
import PropTypes from 'prop-types';

import { selectors } from 'reducers';

import NavigationPanel from './NavigationPanel';

const navLinks = [
    {
        text: 'Dashboard',
        to: '/main/dashboard',
        renderIcon: () => <Icon.BarChart2 className="h-4 w-4 mb-1" />
    },
    {
        text: 'Network',
        to: '/main/network',
        renderIcon: () => <Icon.Share2 className="h-4 w-4 mb-1" />
    },
    {
        text: 'Violations',
        to: '/main/violations',
        renderIcon: () => <Icon.AlertTriangle className="h-4 w-4 mb-1" />
    },
    {
        text: 'Compliance',
        id: 'joe',
        to: '',
        renderIcon: () => <Icon.CheckSquare className="h-4 w-4 mb-1" />,
        panelType: 'compliance'
    },
    {
        text: 'Risk',
        to: '/main/risk',
        renderIcon: () => <Icon.ShieldOff className="h-4 w-4 mb-1" />
    },
    {
        text: 'Images',
        to: '/main/images',
        renderIcon: () => <Icon.FileMinus className="h-4 w-4 mb-1" />
    },
    {
        text: 'Secrets',
        to: '/main/secrets',
        renderIcon: () => <Icon.Lock className="h-4 w-4 mb-1" />
    },
    {
        text: 'Configure',
        to: '',
        renderIcon: () => <Icon.Settings className="h-4 w-4 mb-1" />,
        panelType: 'configure'
    }
];

class LeftNavigation extends Component {
    static propTypes = {
        location: ReactRouterPropTypes.location.isRequired,
        metadata: PropTypes.shape({ version: PropTypes.string })
    };

    static defaultProps = {
        metadata: {
            version: 'latest'
        }
    };

    constructor(props) {
        super(props);
        this.state = {
            panelType: null,
            clickOnPanelItem: false,
            selectedPanel: ''
        };
    }

    componentDidMount() {
        window.onpopstate = e => {
            const url = e.srcElement.location.pathname;
            const link = find(navLinks, navLink => url === navLink.to);
            if (this.state.panelType || link) {
                this.setState({ panelType: null });
            }
        };
    }

    getActiveClassName = navLink => {
        const { pathname } = this.props.location;
        const navText = navLink.text.toLowerCase();
        if (pathname.includes('compliance') && navText === 'compliance') {
            return 'text-base-100 bg-primary-700 hover:bg-primary-700';
        }

        if (
            (pathname.includes('policies') || pathname.includes('integrations')) &&
            navText === 'configure'
        ) {
            return 'text-base-100 bg-primary-700 hover:bg-primary-700';
        }

        if (navLink.to !== '') {
            return 'text-base-100 bg-primary-700 hover:bg-primary-700';
        }
        if (navLink.to === '') {
            if (this.state.panelType && this.state.panelType === navLink.panelType) {
                return 'text-base-100 bg-base-800 hover:bg-base-800';
            }
            if (
                !this.state.panelType &&
                this.state.clickOnPanelItem &&
                this.state.selectedPanel === navText
            ) {
                return 'text-base-100 bg-base-800 hover:bg-base-800';
            }
            return 'bg-primary-800';
        }
        return '';
    };

    closePanel = (clickOnPanelItem, selectedPanel) => () => {
        if (clickOnPanelItem) this.setState({ clickOnPanelItem, selectedPanel });
        this.setState({ panelType: null });
    };

    showNavigationPanel = navLink => e => {
        if (navLink.panelType && this.state.panelType !== navLink.panelType) {
            e.preventDefault();
            this.setState({ panelType: navLink.panelType });
        } else {
            if (this.state.panelType === navLink.panelType) {
                e.preventDefault();
            }
            this.setState({ panelType: null, clickOnPanelItem: false });
        }
    };

    renderLink = (navLink, i, arr) => (
        <Link
            to={navLink.to}
            id={navLink.id}
            activeClassName={this.getActiveClassName(navLink)}
            onClick={this.showNavigationPanel(navLink)}
            className={`flex flex-col font-condensed font-700 border-primary-900 text-primary-400 px-3 no-underline justify-center h-18 hover:bg-base-700 items-center ${
                i === arr.length - 1 ? 'border-b border-b' : 'border-b'
            }`}
        >
            <div className="text-center pb-1">{navLink.renderIcon()}</div>
            <div className="text-center text-base-100">{navLink.text}</div>
        </Link>
    );

    renderLeftSideNavLinks = () => (
        <ul className="flex flex-col list-reset uppercase text-sm tracking-wide">
            {navLinks.map((navLink, i, arr) => (
                <li key={navLink.text}>{this.renderLink(navLink, i, arr)}</li>
            ))}
        </ul>
    );

    renderVersion = () => (
        <div className="flex text-center text-xs font-700">
            <span className="left-navigation p-3 text-primary-400 word-break-all">
                v{this.props.metadata.version}
            </span>
        </div>
    );

    renderNavigationPanel = () => {
        if (!this.state.panelType) return '';
        return <NavigationPanel panelType={this.state.panelType} onClose={this.closePanel} />;
    };

    render() {
        return (
            <div className="flex flex-col justify-between bg-primary-800 flex-none overflow-overlay z-20">
                <nav className="left-navigation">{this.renderLeftSideNavLinks()}</nav>
                {this.renderVersion()}
                {this.renderNavigationPanel()}
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    metadata: selectors.getMetadata
});

export default withRouter(connect(mapStateToProps)(LeftNavigation));
