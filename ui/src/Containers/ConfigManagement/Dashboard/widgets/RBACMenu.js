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

const RBACMenu = ({ match, location, history }) => {
    function handleNavDropdownChange(entityType) {
        const url = URLService.getURL(match, location)
            .base(entityType)
            .url();
        history.push(url);
    }

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
    );
};

RBACMenu.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired
};

export default withRouter(RBACMenu);
