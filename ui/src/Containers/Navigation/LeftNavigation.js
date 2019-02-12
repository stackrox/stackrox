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

const linkClassName =
    'flex flex-col font-condensed font-700 border-primary-900 text-primary-400 px-3 no-underline justify-center h-18 hover:bg-base-700 items-center border-b';
const iconClassName = 'h-4 w-4 mb-1';
const navLinks = [
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
        const baseActiveClass = 'text-base-100 bg-primary-700 hover:bg-primary-700';

        if (
            (pathname.includes('policies') ||
                pathname.includes('integrations') ||
                pathname.includes('access')) &&
            navText === 'configure'
        ) {
            return baseActiveClass;
        }

        if (navLink.to !== '') {
            return baseActiveClass;
        }
        if (navLink.to === '') {
            const baseFocusClass = 'text-base-100 bg-base-800 hover:bg-base-800';
            if (this.state.panelType && this.state.panelType === navLink.panelType) {
                return baseFocusClass;
            }
            if (
                !this.state.panelType &&
                this.state.clickOnPanelItem &&
                this.state.selectedPanel === navText
            ) {
                return baseFocusClass;
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

    renderLink = navLink => (
        <Link
            to={navLink.to}
            activeClassName={this.getActiveClassName(navLink)}
            onClick={this.showNavigationPanel(navLink)}
            className={linkClassName}
        >
            <div className="text-center pb-1">{navLink.renderIcon()}</div>
            <div className="text-center text-base-100">{navLink.text}</div>
        </Link>
    );

    renderLeftSideNavLinks = () => (
        <ul className="flex flex-col list-reset uppercase text-sm tracking-wide">
            {navLinks.map(navLink => (
                <li key={navLink.text}>{this.renderLink(navLink)}</li>
            ))}
        </ul>
    );

    renderFooter = () => (
        <div
            className="flex flex-col flex-none text-center text-xs font-700"
            data-test-id="nav-footer"
        >
            <Link
                to="/main/apidocs"
                className={`${linkClassName} border-t`}
                onClick={this.closePanel()}
            >
                <div className="text-center pb-1">
                    <Icon.HelpCircle className={`${iconClassName} text-primary-400`} />
                </div>
                <div className="text-center text-base-100 font-condensed uppercase text-sm tracking-wide">
                    API Docs
                </div>
            </Link>
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
            <div className="flex flex-col justify-between bg-primary-800 flex-none overflow-overlay z-40">
                <nav className="left-navigation">{this.renderLeftSideNavLinks()}</nav>
                {this.renderFooter()}
                {this.renderNavigationPanel()}
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    metadata: selectors.getMetadata
});

export default withRouter(connect(mapStateToProps)(LeftNavigation));
