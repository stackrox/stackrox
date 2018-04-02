import React, { Component } from 'react';
import * as Icon from 'react-feather';
import Logo from 'Components/icons/logo';
import fetchSummary from 'services/SummaryService';
import AuthService from 'services/AuthService';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';

class TopNavigation extends Component {
    static propTypes = {
        history: ReactRouterPropTypes.history.isRequired
    };

    constructor(props) {
        super(props);
        this.state = {
            counts: null
        };
    }

    componentDidMount() {
        this.getCounts();
    }

    getCounts = () =>
        fetchSummary().then(response => {
            this.setState({ counts: response.data });
        });

    renderLogoutButton = () => {
        if (!AuthService.isLoggedIn()) return '';
        const logout = () => () => {
            AuthService.logout();
            this.props.history.push('/login');
        };
        return (
            <button
                onClick={logout()}
                className="flex-end border-l border-base-300 px-4 no-underline py-3 text-base-600 hover:text-primary-600 items-center"
            >
                <Icon.LogOut className="h-4 w-4 mr-3" />
                <span className="uppercase text-sm">Logout</span>
            </button>
        );
    };

    renderSearchButton = () => (
        <button className="flex-end border-l border-base-300 px-4 no-underline py-3 text-base-600 hover:text-primary-600 items-center">
            <Icon.Search className="h-4 w-4 mr-3 text-center" />
            <span className="uppercase text-sm">Search</span>
        </button>
    );

    renderSummaryCounts = () => {
        if (!this.state.counts) return '';
        return (
            <ul className="flex uppercase text-sm p-0">
                {Object.keys(this.state.counts).map(key => (
                    <li
                        key={key}
                        className="flex flex-col border-r border-base-300 px-4 no-underline py-3 text-base-500 items-center"
                    >
                        <div className="text-xl">{this.state.counts[key]}</div>
                        <div className="text-sm pt-1">{key.replace('num', '')}</div>
                    </li>
                ))}
            </ul>
        );
    };

    render() {
        return (
            <nav className="top-navigation flex flex-row flex-1 justify-between bg-base-100 border-base-300 border-b">
                <div className="flex">
                    <div className="py-2 px-6 border-r bg-white border-base-400">
                        <Logo className="fill-current text-primary-800 h-10 w-10 " />
                    </div>
                    {this.renderSummaryCounts()}
                </div>
                <div className="flex">{this.renderLogoutButton()}</div>
            </nav>
        );
    }
}

export default withRouter(TopNavigation);
