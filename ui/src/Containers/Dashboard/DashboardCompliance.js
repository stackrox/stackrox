import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import sortBy from 'lodash/sortBy';
import { AGGREGATED_RESULTS } from 'queries/controls';
import URLService from 'modules/URLService';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import entityTypes from 'constants/entityTypes';

import Loader from 'Components/Loader';
import Query from 'Components/ThrowingQuery';

const standardsResultsMap = {
    passing: 'var(--success-400)',
    failing: 'var(--alert-400)'
};

class DashboardCompliance extends Component {
    processData = data => {
        if (!data || !data.results || !data.results.results.length) return [];
        const { complianceStandards } = data;
        const modifiedData = data.results.results.map(result => {
            const standard = complianceStandards.find(cs => cs.id === result.aggregationKeys[0].id);
            const { numPassing, numFailing } = result;
            const percentagePassing =
                Math.round((numPassing / (numFailing + numPassing)) * 100) || 0;
            const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
                entityType: standard.id
            });
            const modifiedResult = {
                name: standard.name,
                passing: percentagePassing,
                failing: 100 - percentagePassing,
                link: link.url
            };
            return modifiedResult;
        });
        return sortBy(modifiedData, [datum => datum.name]);
    };

    renderStandardsData = standards =>
        standards.map(standard => {
            const standardResults = ['passing', 'failing'];
            return (
                <div className="pb-3 flex w-full items-center" key={standard.name}>
                    <Link
                        className="text-sm text-primary-700 hover:text-primary-800 tracking-wide underline w-1/3 text-left"
                        to={standard.link}
                    >
                        {standard.name}
                    </Link>

                    <div className="flex flex-1 w-1/2 h-2">
                        {standardResults.map(standardResult => {
                            const resultValue = standard[standardResult];
                            const backgroundStyle = {
                                backgroundColor: standardsResultsMap[standardResult],
                                width: `${resultValue}%`
                            };
                            return (
                                <div
                                    className="border-r border-base-100"
                                    style={backgroundStyle}
                                    key={`${standard.name}-${standardResult}`}
                                />
                            );
                        })}
                    </div>
                </div>
            );
        });

    renderLegend = () =>
        Object.keys(standardsResultsMap).map(result => {
            const backgroundStyle = {
                backgroundColor: standardsResultsMap[result]
            };
            return (
                <div className="flex items-center" key={result}>
                    <div className="h-1 w-8 mr-4" style={backgroundStyle} />
                    <div className="text-sm text-primary-800 tracking-wide capitalize mr-4">
                        {result}
                    </div>
                </div>
            );
        });

    renderScanButton = () => {
        const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.DASHBOARD, {});
        return (
            <div className="h-full">
                <h2 className="bg-base-100 inline-block leading-normal mb-4 p-3 pb-2 pl-6 pr-4 rounded-r-full text-base-600 text-lg text-primary-800 tracking-wide tracking-widest uppercase">
                    Compliance
                </h2>
                <div className="flex flex-col text-center font-700 items-center px-6">
                    <div className="flex flex-col p-4">
                        <span className="mb-4">
                            No Standard results available. Run a scan on the Compliance page.
                        </span>
                        <Link
                            to={link.url}
                            className="no-underline self-center bg-primary-600 px-5 py-3 text-base-100 font-600 rounded-sm uppercase text-sm hover:bg-primary-700"
                        >
                            Go to Compliance
                        </Link>
                    </div>
                </div>
            </div>
        );
    };

    render() {
        return (
            <Query
                query={AGGREGATED_RESULTS}
                variables={{
                    unit: entityTypes.CONTROL,
                    groupBy: [entityTypes.STANDARD]
                }}
                pollInterval={5000}
            >
                {({ loading, data }) => {
                    if (loading) return <Loader />;
                    const results = this.processData(data);
                    if (!results.length) return <div>{this.renderScanButton()}</div>;
                    return (
                        <div>
                            <h2 className="bg-base-100 inline-block leading-normal mb-4 p-3 pb-2 pl-6 pr-4 rounded-r-full text-base-600 text-lg text-primary-800 tracking-wide tracking-widest uppercase self-center">
                                Compliance
                            </h2>
                            <div className="pt-4 px-6">{this.renderStandardsData(results)}</div>
                            <div className="flex flex-1 justify-end w-full pt-4 px-2">
                                {this.renderLegend()}
                            </div>
                        </div>
                    );
                }}
            </Query>
        );
    }
}

export default DashboardCompliance;
