import React from 'react';
import PropTypes from 'prop-types';
import PageHeader from 'Components/PageHeader';
import URLService from 'modules/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import { LIST_STANDARD } from 'queries/standard';
import pluralize from 'pluralize';
import { standardLabels } from 'messages/standards';

import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';
import queryService from 'modules/queryService';
import Menu from 'Components/Menu';
import { ChevronDown } from 'react-feather';
import ExportButton from 'Components/ExportButton';
import PoliciesHeaderTile from './Widgets/PoliciesHeaderTile';

const getLabel = entityType => pluralize(entityLabels[entityType]);

const createTableRows = data => {
    if (!data || !data.results || !data.results.results.length) return [];

    let standardKeyIndex = 0;
    let controlKeyIndex = 0;
    let nodeKeyIndex = 0;
    data.results.results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === entityTypes.STANDARD) standardKeyIndex = idx;
        if (scope === entityTypes.CONTROL) controlKeyIndex = idx;
        if (scope === entityTypes.NODE) nodeKeyIndex = idx;
    });
    const controls = {};
    data.results.results.forEach(({ keys, numFailing }) => {
        if (!keys[controlKeyIndex]) return;
        const controlId = keys[controlKeyIndex].id;
        if (controls[controlId]) {
            controls[controlId].nodes.push(keys[nodeKeyIndex].name);
            if (numFailing) {
                controls[controlId].passing = false;
            }
        } else {
            controls[controlId] = {
                id: controlId,
                standard: standardLabels[keys[standardKeyIndex].id],
                control: `${keys[controlKeyIndex].name} - ${keys[controlKeyIndex].description}`,
                passing: !numFailing,
                nodes: [keys[nodeKeyIndex].name]
            };
        }
    });
    return Object.values(controls);
};

function processControlsData(data) {
    const controls = createTableRows(data);
    const hasFailingControls = controls.some(control => !control.passing);

    return { numControls: controls.length, hasFailingControls };
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
                    query={LIST_STANDARD}
                    variables={{
                        where: queryService.objectToWhereClause({ Standard: 'CIS' }),
                        groupBy: [entityTypes.STANDARD, entityTypes.CONTROL, entityTypes.NODE]
                    }}
                >
                    {({ loading, data }) => {
                        const { numControls, hasFailingControls } = processControlsData(data);
                        return (
                            <TileLink
                                value={numControls}
                                isError={hasFailingControls}
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
                    buttonClass="bg-base-100 hover:bg-base-200 border border-base-400 btn flex font-condensed h-full text-base-600 w-full"
                    buttonContent={
                        <div className="flex items-center text-left px-1">
                            Application & Infrastructure
                            <ChevronDown className="pointer-events-none" />
                        </div>
                    }
                    options={AppMenuOptions}
                />
                <Menu
                    className="w-32"
                    buttonClass="bg-base-100 hover:bg-base-200 border border-base-400 btn flex font-condensed h-full text-base-600 w-full"
                    buttonContent={
                        <div className="flex items-center text-left px-1">
                            RBAC Visibility & Configuration
                            <ChevronDown className="pointer-events-none" />
                        </div>
                    }
                    options={RBACMenuOptions}
                />
                <div className="self-center">
                    <ExportButton
                        fileName="Config Mangement Dashboard"
                        type={null}
                        page="configManagement"
                        pdfId="capture-dashboard"
                    />
                </div>
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
