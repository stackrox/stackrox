import React, { Component } from 'react';
import { NavLink as Link, withRouter } from 'react-router-dom';
import PropTypes from 'prop-types';

const navLinks = [
    {
        text: 'System Policies',
        to: '/main/policies'
    },
    {
        text: 'Integrations',
        to: '/main/integrations'
    },
    {
        text: 'Access Control',
        to: '/main/access'
    }
];

class NavigationPanel extends Component {
    static propTypes = {
        panelType: PropTypes.string.isRequired,
        onClose: PropTypes.func.isRequired
    };

    constructor(props) {
        super(props);
        this.panels = {
            configure: this.renderConfigurePanel
        };
    }

    handleKeyDown = () => {};

    renderConfigurePanel = () => (
        <ul className="flex flex-col overflow-auto list-reset uppercase tracking-wide bg-primary-800 border-r border-l border-primary-900">
            <li className="border-b-2 border-primary-500 px-1 py-5 pl-2 pr-2 text-base-100 font-700">
                Configure StackRox Settings
            </li>
            {navLinks.map(navLink => (
                <li key={navLink.text} className="text-sm">
                    <Link
                        to={navLink.to}
                        onClick={this.props.onClose(true, 'configure')}
                        className="block no-underline text-base-100 px-1 font-700 border-b py-5 border-primary-900 pl-2 pr-2 hover:bg-base-700"
                    >
                        {navLink.text}
                    </Link>
                </li>
            ))}
        </ul>
    );

    render() {
        return (
            <div className="navigation-panel w-full flex">
                {this.panels[this.props.panelType]()}
                <div
                    role="button"
                    tabIndex="0"
                    className="flex-1 opacity-50 bg-primary-700"
                    onClick={this.props.onClose(true)}
                    onKeyDown={this.handleKeyDown}
                />
            </div>
        );
    }
}

export default withRouter(NavigationPanel);
