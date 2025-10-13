import React, { useContext } from 'react';
import PropTypes from 'prop-types';

import workflowStateContext from 'Containers/workflowStateContext';
import { useCaseShortLabels } from 'messages/useCase';
import { createOptions } from 'utils/workflowUtils';

import Menu from 'Components/Menu';

const EntitiesMenu = ({ text, options }) => {
    const workflowState = useContext(workflowStateContext);

    const buttonClass =
        'bg-base-100 hover:bg-primary-200 border-base-400 flex h-full text-base-600 pl-2 border-l border-dashed';

    const dashboardOption = {
        label: `${useCaseShortLabels[workflowState.useCase]} Dashboard`,
        link: workflowState.clear().toUrl(),
    };
    const formattedOptions = [dashboardOption, ...createOptions(options, workflowState)];

    return (
        <Menu
            className="min-w-32 h-full"
            buttonClass={buttonClass}
            buttonText={text}
            options={formattedOptions}
        />
    );
};

EntitiesMenu.propTypes = {
    text: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(PropTypes.string).isRequired,
};

export default EntitiesMenu;
