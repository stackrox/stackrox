import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import pluralize from 'pluralize';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';

import Menu from 'Components/Menu';
import { ChevronDown } from 'react-feather';

const getLabel = entityType => pluralize(entityLabels[entityType]);

const AppMenu = ({ match, location, history }) => {
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

    return (
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
    );
};

AppMenu.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired
};

export default withRouter(AppMenu);
