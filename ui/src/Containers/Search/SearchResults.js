import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import NoResultsMessage from 'Components/NoResultsMessage';
import ReactRowSelectTable from 'Components/ReactRowSelectTable';

import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import PropTypes from 'prop-types';
import capitalize from 'lodash/capitalize';
import lowerCase from 'lodash/lowerCase';
import globalSearchEmptyState from 'images/globalSearchEmptyState.svg';
import { addSearchModifier, addSearchKeyword } from 'utils/searchUtils';

const defaultTabs = [
    {
        text: 'All',
        category: '',
        disabled: false
    },
    {
        text: 'Violations',
        category: 'ALERTS',
        disabled: false
    },
    {
        text: 'Policies',
        category: 'POLICIES',
        disabled: false
    },
    {
        text: 'Deployments',
        category: 'DEPLOYMENTS',
        disabled: false
    },
    {
        text: 'Images',
        category: 'IMAGES',
        disabled: false
    },
    {
        text: 'Secrets',
        category: 'SECRETS',
        disabled: false
    }
];

const mapping = {
    IMAGES: {
        filterOn: ['RISK', 'VIOLATIONS'],
        viewOn: ['IMAGES'],
        name: 'Image Name'
    },
    DEPLOYMENTS: {
        filterOn: ['VIOLATIONS', 'ENVIRONMENT'],
        viewOn: ['RISK'],
        name: 'Deployment Name'
    },
    POLICIES: {
        filterOn: ['VIOLATIONS'],
        viewOn: ['POLICIES'],
        name: 'Policy Name'
    },
    ALERTS: {
        filterOn: [],
        viewOn: ['VIOLATIONS'],
        name: 'Policy Name'
    },
    SECRETS: {
        filterOn: ['RISK'],
        viewOn: ['SECRETS'],
        name: 'Secret Name'
    }
};

const filterOnMapping = {
    RISK: 'DEPLOYMENTS',
    VIOLATIONS: 'ALERTS',
    ENVIRONMENT: 'ENVIRONMENT'
};

class SearchResults extends Component {
    static propTypes = {
        onClose: PropTypes.func.isRequired,
        globalSearchResults: PropTypes.arrayOf(PropTypes.object).isRequired,
        globalSearchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setGlobalSearchCategory: PropTypes.func.isRequired,
        passthroughGlobalSearchOptions: PropTypes.func.isRequired,
        tabs: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        defaultTab: PropTypes.shape({})
    };

    static defaultProps = {
        defaultTab: null
    };

    onTabClick = tab => {
        this.props.setGlobalSearchCategory(tab.category);
    };

    onLinkHandler = (searchCategory, category, toURL, name) => () => {
        let searchOptions = this.props.globalSearchOptions.slice();
        if (name) {
            searchOptions = addSearchModifier(searchOptions, `${mapping[searchCategory].name}:`);
            searchOptions = addSearchKeyword(searchOptions, name);
        }
        this.props.passthroughGlobalSearchOptions(searchOptions, category);
        this.props.onClose(toURL);
    };

    renderTabs = () => {
        const { tabs } = this.props;
        return (
            <section className="flex flex-auto h-full">
                <div className="flex flex-1">
                    <Tabs
                        className="bg-white mb-8"
                        headers={tabs}
                        onTabClick={this.onTabClick}
                        default={this.props.defaultTab}
                        tabClass="tab flex-1 items-center justify-center font-600"
                        tabActiveClass="tab flex-1 items-center justify-center border-b-2 border-primary-400 font-700 text-primary-500"
                        tabDisabledClass="tab flex-1 items-center justify-center font-600 disabled"
                        tabContentBgColor="bg-white"
                    >
                        {tabs.map(tab => (
                            <TabContent key={tab.text}>
                                <div
                                    key={tab.text}
                                    className="flex flex-1 w-full h-full pl-3 pr-3 pt-3 rounded-sm"
                                >
                                    {this.renderTable(tab.text)}
                                </div>
                            </TabContent>
                        ))}
                    </Tabs>
                </div>
            </section>
        );
    };

    renderTable = () => {
        const columns = [
            {
                accessor: 'name',
                Header: 'Name',
                Cell: ({ original }) => (
                    <div className="flex flex-col">
                        <div>{original.name}</div>
                        {original.location ? (
                            <div className="text-primary-300 italic text-sm pt-2">
                                in {original.location}
                            </div>
                        ) : null}
                    </div>
                )
            },
            {
                accessor: 'category',
                Header: 'Type',
                Cell: ({ original }) => capitalize(original.category)
            },
            {
                Header: 'View On:',
                Cell: ({ original }) => (
                    <ul className="p-0 list-reset flex flex-row">
                        {!mapping[original.category] || !mapping[original.category].viewOn ? (
                            <li className="text-base-400">N/A</li>
                        ) : (
                            mapping[original.category].viewOn.map((item, index) => (
                                <li key={index}>
                                    <button
                                        onClick={this.onLinkHandler(
                                            original.category,
                                            item,
                                            `/main/${lowerCase(item)}/${original.id}`
                                        )}
                                        className="inline-block py-1 px-2 no-underline text-center uppercase bg-primary-100 border-2 border-base-200 mr-1 rounded-sm text-sm text-base-600"
                                    >
                                        {item}
                                    </button>
                                </li>
                            ))
                        )}
                    </ul>
                ),
                sortable: false
            },
            {
                Header: 'Filter On:',
                Cell: ({ original }) => (
                    <ul className="p-0 list-reset flex flex-row">
                        {!mapping[original.category] || !mapping[original.category].filterOn ? (
                            <li className="text-base-400">N/A</li>
                        ) : (
                            mapping[original.category].filterOn.map((item, index) => (
                                <li key={index}>
                                    <button
                                        onClick={this.onLinkHandler(
                                            original.category,
                                            filterOnMapping[item],
                                            `/main/${lowerCase(item)}`,
                                            original.name
                                        )}
                                        className="inline-block py-1 px-2 no-underline text-center uppercase bg-primary-100 border-2 border-base-200 mr-1 rounded-sm text-sm text-base-600"
                                    >
                                        {item}
                                    </button>
                                </li>
                            ))
                        )}
                    </ul>
                ),
                sortable: false
            }
        ];
        const rows = this.props.globalSearchResults;
        if (!rows.length) return <NoResultsMessage message="No Search Results." />;
        return <ReactRowSelectTable rows={rows} columns={columns} noDataText="No Search Results" />;
    };

    render() {
        if (!this.props.globalSearchOptions.length) {
            return (
                <div className="bg-white flex flex-1 items-center justify-center">
                    <img
                        src={globalSearchEmptyState}
                        className="flex h-full w-1/2"
                        alt="No search results"
                    />
                </div>
            );
        }
        return (
            <div className="bg-white flex-1">
                <h1 className="w-full text-xl text-primary-600 px-4 py-6 font-400">
                    {this.props.globalSearchResults.length} search results
                </h1>
                {this.renderTabs()}
            </div>
        );
    }
}

const getTabs = createSelector([selectors.getGlobalSearchCounts], globalSearchCounts => {
    if (globalSearchCounts.length === 0) return defaultTabs;

    const newTabs = [];
    defaultTabs.forEach(tab => {
        const newTab = Object.assign({}, tab);
        const currentTab = globalSearchCounts.find(obj => obj.category === tab.category);
        if (currentTab) {
            newTab.text += ` (${currentTab.count})`;
            if (currentTab.count === '0') newTab.disabled = true;
        }
        newTabs.push(newTab);
    });
    return newTabs;
});

const getDefaultTab = createSelector([selectors.getGlobalSearchCategory], globalSearchCategory => {
    const tab = defaultTabs.find(obj => obj.category === globalSearchCategory);
    return tab;
});

const mapStateToProps = createStructuredSelector({
    globalSearchResults: selectors.getGlobalSearchResults,
    globalSearchOptions: selectors.getGlobalSearchOptions,
    tabs: getTabs,
    defaultTab: getDefaultTab
});

const mapDispatchToProps = dispatch => ({
    setGlobalSearchCategory: category =>
        dispatch(globalSearchActions.setGlobalSearchCategory(category)),
    passthroughGlobalSearchOptions: (searchOptions, category) =>
        dispatch(globalSearchActions.passthroughGlobalSearchOptions(searchOptions, category))
});

export default connect(mapStateToProps, mapDispatchToProps)(SearchResults);
