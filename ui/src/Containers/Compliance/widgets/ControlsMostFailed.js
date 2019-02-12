import React from 'react';
import PropTypes from 'prop-types';
import LinkListWidget from 'Components/LinkListWidget';
import pageTypes from 'constants/pageTypes';
import URLService from 'modules/URLService';
import pluralize from 'pluralize';
import entityTypes, { resourceTypes } from 'constants/entityTypes';
import contextTypes from 'constants/contextTypes';
import { resourceLabels } from 'messages/common';
import { AGGREGATED_RESULTS_WITH_CONTROLS as QUERY } from 'queries/controls';
import queryService from 'modules/queryService';

const ControlsMostFailed = ({ params, limit }) => {
    const { entityType, query } = params;

    const isResource = !!resourceTypes[entityType];
    const groupBy = [entityTypes.CONTROL, entityTypes.STANDARD];
    if (isResource) groupBy.push(entityType);
    const variables = {
        groupBy,
        unit: entityTypes.CONTROL,
        where: queryService.objectToWhereClause(query)
    };

    function processData(data) {
        if (!data || !data.results || !data.results.results || !data.results.results.length)
            return [];

        const { results } = data.results;
        const { complianceStandards } = data;
        const controlNameLookup = complianceStandards.reduce(
            (acc, standard) => acc.concat(standard.controls),
            []
        );

        let ctrlIndex;
        let standardIndex;

        results[0].aggregationKeys.forEach((item, i) => {
            if (item.scope === entityTypes.CONTROL) ctrlIndex = i;
            else if (item.scope === entityTypes.STANDARD) standardIndex = i;
        });

        const totals = results
            .filter(item => item.numPassing > 0 || item.numFailing > 0)
            .reduce((acc, result) => {
                const ctrlId = result.aggregationKeys[ctrlIndex].id;
                const standardId = result.aggregationKeys[standardIndex].id;
                if (acc[ctrlId]) {
                    acc[ctrlId].totalFailing += result.numFailing;
                } else {
                    acc[ctrlId] = {
                        totalFailing: result.numFailing,
                        standardId
                    };
                }
                return acc;
            }, {});

        return Object.entries(totals)
            .sort((a, b) => b[1].totalFailing - a[1].totalFailing)
            .map(entry => {
                const control = controlNameLookup.find(ctrl => ctrl.id === entry[0]);
                const label = control ? `${control.name} - ${control.description}` : '';
                const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.ENTITY, {
                    entityId: entry[0],
                    entityType: entry[1].standardId
                }).url;
                return { label, link };
            });
    }

    function getHeadline() {
        const titleEntity = isResource
            ? `across ${pluralize(resourceLabels[entityType])}`
            : 'in this standard';
        return `Controls most failed ${titleEntity}`;
    }

    return (
        <LinkListWidget
            query={QUERY}
            variables={variables}
            processData={processData}
            getHeadline={getHeadline}
            limit={limit}
        />
    );
};

ControlsMostFailed.propTypes = {
    params: PropTypes.shape({
        entityType: PropTypes.string,
        context: PropTypes.string,
        query: PropTypes.shape({})
    }).isRequired,
    limit: PropTypes.number
};

ControlsMostFailed.defaultProps = {
    limit: 10
};

export default ControlsMostFailed;
