import React from 'react';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import { resourceLabels } from 'messages/common';
import { standardLabels } from 'messages/standards';
import URLService from 'modules/URLService';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import pluralize from 'pluralize';
import toLower from 'lodash/toLower';

import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import HorizontalBarChart from 'Components/visuals/HorizontalBar';
import NoResultsMessage from 'Components/NoResultsMessage';
import { AGGREGATED_RESULTS as QUERY } from 'queries/controls';

function formatAsPercent(x) {
    return `${x}%`;
}

function processData(data, type) {
    if (!data || !data.results || !data.results.results.length) return [];
    const { complianceStandards } = data;
    const standardsMapping = {};

    data.results.results.forEach(result => {
        const standardId = result.aggregationKeys[0].id;
        const { numPassing, numFailing } = result;
        if (!standardsMapping[standardId]) {
            standardsMapping[standardId] = {
                passing: numPassing,
                total: numPassing + numFailing
            };
        } else {
            standardsMapping[standardId] = {
                passing: standardsMapping[standardId].passing + numPassing,
                total: standardsMapping[standardId].total + (numPassing + numFailing)
            };
        }
    });

    const barData = Object.keys(standardsMapping).map(standardId => {
        const standard = complianceStandards.find(cs => cs.id === standardId);
        const { passing, total } = standardsMapping[standardId];
        const percentagePassing = Math.round((passing / total) * 100) || 0;
        const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
            entityType: entityTypes.CONTROL,
            query: {
                Standard: standardLabels[standard.id]
            }
        });
        const dataPoint = {
            y: standardBaseTypes[standardId],
            x: percentagePassing,
            hint: {
                title: `${standard.name} Standard - ${percentagePassing}% Passing`,
                body: `${total - passing} failing controls across all ${pluralize(
                    resourceLabels[type]
                )}`
            },
            link: link.url
        };
        return dataPoint;
    });

    return barData;
}

const StandardsAcrossEntity = ({ entityType, bodyClassName, className }) => {
    const variables = {
        groupBy: [entityTypes.STANDARD, entityType],
        unit: entityTypes.CONTROL
    };
    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                let contents;
                const headerText = `Passing standards across ${entityType}s`;
                if (!loading || data.complianceStandards) {
                    const results = processData(data, entityType);
                    if (!results.length) {
                        contents = (
                            <NoResultsMessage message="No data available. Please run a scan." />
                        );
                    } else {
                        contents = (
                            <HorizontalBarChart data={results} valueFormat={formatAsPercent} />
                        );
                    }
                } else {
                    contents = <Loader />;
                }
                return (
                    <Widget
                        className={`s-2 ${className}`}
                        header={headerText}
                        id={`standards-across-${toLower(entityType)}`}
                        bodyClassName={`graph-bottom-border ${bodyClassName}`}
                    >
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

StandardsAcrossEntity.propTypes = {
    entityType: PropTypes.string.isRequired,
    bodyClassName: PropTypes.string,
    className: PropTypes.string
};

StandardsAcrossEntity.defaultProps = {
    bodyClassName: 'px-4 pt-1',
    className: ''
};

export default StandardsAcrossEntity;
