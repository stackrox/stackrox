import React, { useContext } from 'react';
import PropTypes from 'prop-types';

import workflowStateContext from 'Containers/workflowStateContext';
import { entityGroupMap } from 'utils/entityRelationships';
import { useCaseShortLabels } from 'messages/useCase';
import { getOption, createOptions } from 'utils/workflowUtils';

import Menu from 'Components/Menu';

const EntitiesMenu = ({ text, options, grouped }) => {
    const workflowState = useContext(workflowStateContext);

    function createGroupedOptions(types) {
        const groupedOptions = {};
        types.forEach((type) => {
            const option = getOption(type, workflowState);
            const optionGroup = groupedOptions[entityGroupMap[type]];
            if (optionGroup) {
                groupedOptions[entityGroupMap[type]].push(option);
            } else {
                groupedOptions[entityGroupMap[type]] = [option];
            }
        });
        return groupedOptions;
    }

    const buttonClass =
        'bg-base-100 hover:bg-primary-200 border-base-400 font-weight-600 uppercase font-condensed flex h-full text-base-600 pl-2 border-l border-dashed text-sm';

    let formattedOptions = [];
    if (!grouped) {
        const dashboardOption = {
            label: `${useCaseShortLabels[workflowState.useCase]} Dashboard`,
            link: workflowState.clear().toUrl(),
        };
        formattedOptions = [dashboardOption, ...createOptions(options, workflowState)];
    } else {
        formattedOptions = createGroupedOptions(options);
    }

    return (
        <Menu
            className="min-w-32 h-full"
            menuClassName={grouped ? 'bg-primary-200 min-w-48' : ''}
            buttonClass={buttonClass}
            buttonText={text}
            options={formattedOptions}
            grouped={grouped}
        />
    );
};

EntitiesMenu.propTypes = {
    text: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(PropTypes.string).isRequired,
    grouped: PropTypes.bool,
};

EntitiesMenu.defaultProps = {
    grouped: false,
};

export default EntitiesMenu;
