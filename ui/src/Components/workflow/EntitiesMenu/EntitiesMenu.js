import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import entityLabels from 'messages/entity';
import { ChevronDown } from 'react-feather';

import workflowStateContext from 'Containers/workflowStateContext';
import { entityGroupMap } from 'modules/entityRelationships';

import DashboardMenu from 'Components/DashboardMenu';
import Menu from 'Components/Menu';

const getLabel = entityType => pluralize(entityLabels[entityType]);

const EntitiesMenu = ({ text, options, grouped }) => {
    const workflowState = useContext(workflowStateContext);

    function getOption(type) {
        return {
            label: getLabel(type),
            link: workflowState.resetPage(type).toUrl()
        };
    }

    function createOptions(types) {
        return types.map(type => getOption(type));
    }

    function createGroupedOptions(types) {
        const groupedOptions = {};
        types.forEach(type => {
            const option = getOption(type);
            const optionGroup = groupedOptions[entityGroupMap[type]];
            if (optionGroup) {
                groupedOptions[entityGroupMap[type]].push(option);
            } else {
                groupedOptions[entityGroupMap[type]] = [option];
            }
        });
        return groupedOptions;
    }

    if (!grouped) {
        const formattedOptions = createOptions(options);
        return <DashboardMenu text={text} options={formattedOptions} />;
    }
    const formattedGroupedOptions = createGroupedOptions(options);
    return (
        <Menu
            className="h-full"
            menuClassName="bg-primary-200 min-w-48"
            buttonClass="bg-base-100 hover:bg-primary-200 border-l border-dashed border-base-400 font-weight-600 uppercase font-condensed flex font-condensed h-full pl-2 text-base-600"
            buttonContent={
                <div className="flex items-center text-left px-2">
                    {text}
                    <ChevronDown className="pointer-events-none ml-2" />
                </div>
            }
            options={formattedGroupedOptions}
            grouped
        />
    );
};

EntitiesMenu.propTypes = {
    text: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(PropTypes.string).isRequired,
    grouped: PropTypes.bool
};

EntitiesMenu.defaultProps = {
    grouped: false
};

export default EntitiesMenu;
