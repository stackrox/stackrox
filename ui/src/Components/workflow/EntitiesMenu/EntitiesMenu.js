import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import entityLabels from 'messages/entity';
import { ChevronDown } from 'react-feather';

import workflowStateContext from 'Containers/workflowStateContext';

import DashboardMenu from 'Components/DashboardMenu';
import Menu from 'Components/Menu';

const getLabel = entityType => pluralize(entityLabels[entityType]);

const createOptions = (workflowState, types) => {
    return types.map(type => {
        return {
            label: getLabel(type),
            link: workflowState.resetPage(type).toUrl()
        };
    });
};

const EntitiesMenu = ({ text, options, dashboard }) => {
    const workflowState = useContext(workflowStateContext);
    const formattedOptions = createOptions(workflowState, options);
    if (dashboard) return <DashboardMenu text={text} options={formattedOptions} />;
    return (
        <Menu
            className="h-full"
            buttonClass="bg-base-100 hover:bg-base-200 border-l border-dashed border-base-400 font-weight-600 uppercase font-condensed flex font-condensed h-full pl-2 text-base-600"
            buttonContent={
                <div className="flex items-center text-left px-2">
                    {text}
                    <ChevronDown className="pointer-events-none ml-2" />
                </div>
            }
            options={formattedOptions}
        />
    );
};

EntitiesMenu.propTypes = {
    text: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(PropTypes.string).isRequired,
    dashboard: PropTypes.bool
};

EntitiesMenu.defaultProps = {
    dashboard: false
};

export default EntitiesMenu;
