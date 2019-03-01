import React from 'react';
import PropTypes from 'prop-types';
import LinkListWidget from 'Components/LinkListWidget';
import pageTypes from 'constants/pageTypes';
import URLService from 'modules/URLService';
import pluralize from 'pluralize';
import entityTypes, { resourceTypes } from 'constants/entityTypes';
import contextTypes from 'constants/contextTypes';
import { resourceLabels } from 'messages/common';
import { standardLabels } from 'messages/standards';

import { AGGREGATED_RESULTS_WITH_CONTROLS as QUERY } from 'queries/controls';
import queryService from 'modules/queryService';

const ControlsMostFailed = ({ entityType, query, limit, showEmpty }) => {
    const whereClauseValues = { ...query };
    const isResource = !!resourceTypes[entityType];
    const groupBy = [entityTypes.CONTROL, entityTypes.STANDARD];
    if (isResource) {
        groupBy.push(entityType);
    } else {
        whereClauseValues[entityTypes.STANDARD] = standardLabels[entityType];
    }

    const variables = {
        groupBy,
        unit: entityTypes.CONTROL,
        where: queryService.objectToWhereClause(whereClauseValues)
    };

    function processData(data) {
        if (!data || !data.results || !data.results.results || !data.results.results.length)
            return [];

        const { results } = data.results;
        const { complianceStandards } = data;
        const controls = complianceStandards.reduce((acc, standard) => {
            const standardName = standard.name;
            const standardControls = standard.controls.map(control => ({
                id: control.id,
                label: isResource
                    ? `${standardName} - ${control.name}: ${control.description}`
                    : `${control.name}: ${control.description}`
            }));

            return acc.concat(standardControls);
        }, []);

        let ctrlIndex;
        let standardIndex;

        results[0].aggregationKeys.forEach((item, i) => {
            if (item.scope === entityTypes.CONTROL) ctrlIndex = i;
            else if (item.scope === entityTypes.STANDARD) standardIndex = i;
        });

        const totals = results
            .filter(item => item.numPassing > 0 || item.numFailing > 0)
            .reduce((acc, { aggregationKeys, numFailing }) => {
                const ctrlId = aggregationKeys[ctrlIndex].id;
                const standardId = aggregationKeys[standardIndex].id;
                if (acc[ctrlId]) {
                    acc[ctrlId].totalFailing += numFailing;
                } else {
                    acc[ctrlId] = {
                        totalFailing: numFailing,
                        standardId
                    };
                }
                return acc;
            }, {});

        return Object.entries(totals)
            .sort((a, b) => b[1].totalFailing - a[1].totalFailing)
            .map(entry => {
                const control = controls.find(ctrl => ctrl.id === entry[0]);
                const label = control ? control.label : '';
                const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.ENTITY, {
                    entityType: entityTypes.CONTROL,
                    controlId: entry[0],
                    standardId: entry[1].standardId
                }).url;
                return { label, link };
            });
    }

    function getHeadline() {
        const titleEntity = isResource
            ? `across ${pluralize(resourceLabels[entityType])}`
            : `in ${standardLabels[entityType]}`;
        return `Controls most failed ${titleEntity}`;
    }

    return (
        <LinkListWidget
            query={QUERY}
            variables={variables}
            processData={processData}
            getHeadline={getHeadline}
            limit={limit}
            showEmpty={showEmpty}
        />
    );
};

ControlsMostFailed.propTypes = {
    entityType: PropTypes.string,
    query: PropTypes.shape({}),
    limit: PropTypes.number,
    showEmpty: PropTypes.bool
};

ControlsMostFailed.defaultProps = {
    limit: 10,
    showEmpty: false,
    entityType: null,
    query: null
};

export default ControlsMostFailed;
