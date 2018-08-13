import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import Table from 'Components/Table';

import UnderlineTabs from 'Components/UnderlineTabs';
import TabContent from 'Components/TabContent';
import PropTypes from 'prop-types';
import capitalize from 'lodash/capitalize';
import lowerCase from 'lodash/lowerCase';
import globalSearchEmptyState from 'images/globalSearchEmptyState.svg';
import { addSearchModifier, addSearchKeyword } from 'utils/searchUtils';

const tabs = [
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
        filterOn: ['VIOLATIONS'],
        viewOn: ['ENVIRONMENT', 'RISK'],
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
    VIOLATIONS: 'ALERTS'
};

const viewOnFilters = ['ENVIRONMENT'];

class SearchResults extends Component {
    static propTypes = {
        onClose: PropTypes.func.isRequired,
        globalSearchResults: PropTypes.arrayOf(PropTypes.object).isRequired,
        globalSearchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setGlobalSearchCategory: PropTypes.func.isRequired,
        passthroughGlobalSearchOptions: PropTypes.func.isRequired,
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

    renderTabs = () => (
        <section className="flex flex-auto h-full">
            <div className="flex flex-1">
                <UnderlineTabs
                    className="bg-white"
                    headers={tabs}
                    onTabClick={this.onTabClick}
                    default={this.props.defaultTab}
                >
                    {tabs.map(tab => (
                        <TabContent key={tab.text}>
                            <div
                                key={tab.text}
                                className="flex flex-1 w-full p-3 overflow-y-scroll rounded-sm"
                            >
                                {this.renderTable(tab.text)}
                            </div>
                        </TabContent>
                    ))}
                </UnderlineTabs>
            </div>
        </section>
    );

    renderTable = () => {
        if (!this.props.globalSearchResults.length) {
            return (
                <div className="flex flex-1 items-center justify-center bg-white">
                    No Search Results
                </div>
            );
        }
        const table = {
            columns: [
                {
                    keys: ['name', 'location'],
                    label: 'Name',
                    keyValueFunc: (name, location) => (
                        <div className="flex flex-col">
                            <div>{name}</div>
                            {location ? (
                                <div className="text-primary-300 italic text-sm pt-2">
                                    in {location}
                                </div>
                            ) : null}
                        </div>
                    )
                },
                { key: 'category', keyValueFunc: value => capitalize(value), label: 'Type' },
                {
                    keys: ['category', 'id', 'name', 'score'],
                    label: 'View On:',
                    keyValueFunc: (category, id, name) => (
                        <ul className="p-0 list-reset flex flex-row">
                            {!mapping[category] || !mapping[category].viewOn ? (
                                <li className="text-base-400">N/A</li>
                            ) : (
                                mapping[category].viewOn.map((item, index) => (
                                    <li key={index}>
                                        <button
                                            onClick={this.onLinkHandler(
                                                category,
                                                item,
                                                `/main/${lowerCase(item)}${
                                                    viewOnFilters.includes(item) ? '' : `/${id}`
                                                }`,
                                                viewOnFilters.includes(item) ? name : null
                                            )}
                                            className="inline-block py-1 px-2 no-underline text-center uppercase bg-primary-100 border-2 border-base-200 mr-1 rounded-sm text-sm text-base-600"
                                        >
                                            {item}
                                        </button>
                                    </li>
                                ))
                            )}
                        </ul>
                    )
                },
                {
                    keys: ['category', 'id', 'name'],
                    label: 'Filter On:',
                    keyValueFunc: (category, id, name) => (
                        <ul className="p-0 list-reset flex flex-row">
                            {!mapping[category] || !mapping[category].filterOn ? (
                                <li className="text-base-400">N/A</li>
                            ) : (
                                mapping[category].filterOn.map((item, index) => (
                                    <li key={index}>
                                        <button
                                            onClick={this.onLinkHandler(
                                                category,
                                                filterOnMapping[item],
                                                `/main/${lowerCase(item)}`,
                                                name
                                            )}
                                            className="inline-block py-1 px-2 no-underline text-center uppercase bg-primary-100 border-2 border-base-200 mr-1 rounded-sm text-sm text-base-600"
                                        >
                                            {item}
                                        </button>
                                    </li>
                                ))
                            )}
                        </ul>
                    )
                }
            ],
            rows: this.props.globalSearchResults
        };
        return <Table columns={table.columns} rows={table.rows} />;
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

const getDefaultTab = createSelector([selectors.getGlobalSearchCategory], globalSearchCategory => {
    const tab = tabs.find(obj => obj.category === globalSearchCategory);
    return tab;
});

const mapStateToProps = createStructuredSelector({
    globalSearchResults: selectors.getGlobalSearchResults,
    globalSearchOptions: selectors.getGlobalSearchOptions,
    defaultTab: getDefaultTab
});

const mapDispatchToProps = dispatch => ({
    setGlobalSearchCategory: category =>
        dispatch(globalSearchActions.setGlobalSearchCategory(category)),
    passthroughGlobalSearchOptions: (searchOptions, category) =>
        dispatch(globalSearchActions.passthroughGlobalSearchOptions(searchOptions, category))
});

export default connect(mapStateToProps, mapDispatchToProps)(SearchResults);
