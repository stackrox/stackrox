import React from 'react';
import PropTypes from 'prop-types';
import PageHeader from 'Components/PageHeader';
import URLService from 'modules/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import { AGGREGATED_RESULTS_WITH_CONTROLS as CISControlsQuery } from 'queries/controls';
import pluralize from 'pluralize';

import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';
import queryService from 'modules/queryService';
import Menu from 'Components/Menu';
import { ChevronDown } from 'react-feather';
import PoliciesHeaderTile from './Widgets/PoliciesHeaderTile';

const getLabel = entityType => pluralize(entityLabels[entityType]);

function processControlsData(data) {
    let totalControls = 0;
    let hasViolations = false;

    if (!data || !data.results || !data.results.results || !data.results.results.length)
        return { totalControls, hasViolations };

    const { results } = data.results;
    totalControls = data.complianceStandards
        .filter(standard => standard.name.includes('CIS'))
        .reduce((total, standard) => {
            return total + standard.controls.length;
        }, 0);

    hasViolations = !!results.find(({ numFailing }) => {
        return numFailing > 0;
    });

    return { totalControls, hasViolations };
}

const ConfigManagementHeader = ({ match, location, history, classes, bgStyle }) => {
    const controlsLink = URLService.getURL(match, location)
        .base(entityTypes.CONTROL)
        .url();

    function handleNavDropdownChange(entityType) {
        const url = URLService.getURL(match, location)
            .base(entityType)
            .url();
        history.push(url);
    }

    const AppMenuOptions = [
        {
            label: getLabel(entityTypes.CLUSTER),
            onClick: () => handleNavDropdownChange(entityTypes.CLUSTER)
        },
        {
            label: getLabel(entityTypes.NAMESPACE),
            onClick: () => handleNavDropdownChange(entityTypes.NAMESPACE)
        },
        {
            label: getLabel(entityTypes.NODE),
            onClick: () => handleNavDropdownChange(entityTypes.NODE)
        },
        {
            label: getLabel(entityTypes.DEPLOYMENT),
            onClick: () => handleNavDropdownChange(entityTypes.DEPLOYMENT)
        },
        {
            label: getLabel(entityTypes.IMAGE),
            onClick: () => handleNavDropdownChange(entityTypes.IMAGE)
        },
        {
            label: getLabel(entityTypes.SECRET),
            onClick: () => handleNavDropdownChange(entityTypes.SECRET)
        }
    ];

    const RBACMenuOptions = [
        {
            label: getLabel(entityTypes.SUBJECT),
            onClick: () => handleNavDropdownChange(entityTypes.SUBJECT)
        },
        {
            label: getLabel(entityTypes.SERVICE_ACCOUNT),
            onClick: () => handleNavDropdownChange(entityTypes.SERVICE_ACCOUNT)
        },
        {
            label: getLabel(entityTypes.ROLE),
            onClick: () => handleNavDropdownChange(entityTypes.ROLE)
        }
    ];

    return (
        <PageHeader
            classes={classes}
            bgStyle={bgStyle}
            header="Configuration Management"
            subHeader="Dashboard"
        >
            <div className="flex flex-1 justify-end">
                <PoliciesHeaderTile />

                <Query
                    query={CISControlsQuery}
                    variables={{
                        groupBy: entityTypes.CONTROL,
                        unit: entityTypes.CONTROL,
                        where: queryService.objectToWhereClause({ standard: 'CIS' })
                    }}
                >
                    {({ loading, data }) => {
                        const { totalControls, hasViolations } = processControlsData(data);
                        return (
                            <TileLink
                                value={totalControls}
                                isError={hasViolations}
                                caption="CIS Controls"
                                to={controlsLink}
                                loading={loading}
                                className="rounded-none"
                            />
                        );
                    }}
                </Query>
                <Menu
                    className="w-32"
                    buttonClass="bg-base-100 hover:bg-base-200 border border-base-400 btn flex font-condensed h-full text-primary-500 w-full"
                    buttonContent={
                        <div className="flex items-center">
                            Application & Infrastructure
                            <ChevronDown className="pointer-events-none" />
                        </div>
                    }
                    options={AppMenuOptions}
                />
                <Menu
                    className="w-32"
                    buttonClass="bg-base-100 hover:bg-base-200 border border-base-400 btn flex font-condensed h-full text-primary-500 w-full"
                    buttonContent={
                        <div className="flex items-center">
                            RBAC Visibility & Controls
                            <ChevronDown className="pointer-events-none" />
                        </div>
                    }
                    options={RBACMenuOptions}
                />
            </div>
        </PageHeader>
    );
};

ConfigManagementHeader.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    classes: PropTypes.string,
    bgStyle: PropTypes.shape({})
};

ConfigManagementHeader.defaultProps = {
    classes: null,
    bgStyle: null
};

export default withRouter(ConfigManagementHeader);
