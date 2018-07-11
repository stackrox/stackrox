import React, { Component } from 'react';
import * as Icon from 'react-feather';
import { withRouter, NavLink as Link } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import find from 'lodash/find';
import NavigationPanel from './NavigationPanel';

const navLinks = [
    {
        text: 'Dashboard',
        to: '/main/dashboard',
        renderIcon: () => <Icon.BarChart2 className="h-4 w-4 mb-1" />
    },
    {
        text: 'Violations',
        to: '/main/violations',
        renderIcon: () => <Icon.AlertTriangle className="h-4 w-4 mb-1" />
    },
    {
        text: 'Compliance',
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
        location: ReactRouterPropTypes.location.isRequired
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
            return 'text-white bg-primary-600';
        }

        if (
            (pathname.includes('policies') || pathname.includes('integrations')) &&
            navText === 'configure'
        ) {
            return 'text-white bg-primary-600';
        }

        if (navLink.to !== '') {
            return 'text-white bg-primary-600';
        }
        if (navLink.to === '') {
            if (this.state.panelType && this.state.panelType === navLink.panelType) {
                return 'text-white bg-primary-700';
            } else if (
                !this.state.panelType &&
                this.state.clickOnPanelItem &&
                this.state.selectedPanel === navText
            ) {
                return 'text-white bg-primary-600';
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
            activeClassName={this.getActiveClassName(navLink)}
            onClick={this.showNavigationPanel(navLink)}
            className={`flex flex-col font-condensed font-700 border-primary-900 text-primary-400 px-3 no-underline py-4 hover:bg-primary-700 items-center ${
                i === arr.length - 1 ? 'border-b border-t' : 'border-t'
            }`}
        >
            <div className="text-center pb-1">{navLink.renderIcon()}</div>
            <div className="text-center text-white">{navLink.text}</div>
        </Link>
    );

    renderLeftSideNavLinks = () => (
        <ul className="flex flex-col list-reset uppercase text-sm tracking-wide">
            {navLinks.map((navLink, i, arr) => (
                <li key={navLink.text} className="flex-col ">
                    {this.renderLink(navLink, i, arr)}
                </li>
            ))}
        </ul>
    );

    renderNavigationPanel = () => {
        if (!this.state.panelType) return '';
        return <NavigationPanel panelType={this.state.panelType} onClose={this.closePanel} />;
    };

    render() {
        return (
            <div className="flex flex-col bg-primary-800">
                <nav className="left-navigation">{this.renderLeftSideNavLinks()}</nav>
                {this.renderNavigationPanel()}
            </div>
        );
    }
}

export default withRouter(LeftNavigation);
