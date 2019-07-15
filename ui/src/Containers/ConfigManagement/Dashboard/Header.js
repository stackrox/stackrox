import React from 'react';
import PropTypes from 'prop-types';
import PageHeader from 'Components/PageHeader';
import URLService from 'modules/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityTypes from 'constants/entityTypes';
import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';
import { AGGREGATED_RESULTS_WITH_CONTROLS as CISControlsQuery } from 'queries/controls';
import queryService from 'modules/queryService';
import ReactSelect from 'Components/ReactSelect';
import PoliciesHeaderTile from './Widgets/PoliciesHeaderTile';

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
        .base(entityTypes.control)
        .url();

    const AppMenuOptions = [
        { value: entityTypes.SECRET, label: 'Secrets' },
        { value: entityTypes.NAMESPACE, label: 'Namespaces' },
        { value: entityTypes.DEPLOYMENT, label: 'Deployments' },
        { value: entityTypes.IMAGE, label: 'Images' },
        { value: entityTypes.NODE, label: 'Nodes' }
    ];

    const RBACMenuOptions = [
        { value: entityTypes.SUBJECT, label: 'Users & Groups' },
        { value: entityTypes.SERVICE_ACCOUNT, label: 'Service Accounts' },
        { value: entityTypes.ROLE, label: 'Roles' }
    ];

    function handleNavDropdownChange(entityType) {
        const url = URLService.getURL(match, location)
            .base(entityType)
            .url();
        history.push(url);
    }
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
                <ReactSelect
                    options={AppMenuOptions}
                    className="w-32 text-base-600 bg-base-200"
                    placeholder="Application & Infrastructure"
                    onChange={handleNavDropdownChange}
                    styles={{
                        indicatorSeparator: () => ({
                            display: 'none'
                        }),
                        control: () => ({
                            borderLeft: 'none',
                            borderRight: 'none'
                        })
                    }}
                />
                <ReactSelect
                    options={RBACMenuOptions}
                    className="text-base-600 bg-base-200 w-36"
                    placeholder="RBAC Visibility & Configuration"
                    onChange={handleNavDropdownChange}
                    styles={{
                        indicatorSeparator: () => ({
                            display: 'none'
                        })
                    }}
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
