import React from 'react';
import PropTypes from 'prop-types';

import { ChevronDown } from 'react-feather';
import Menu from 'Components/Menu';

const DashboardMenu = ({ text, options }) => {
    return (
        <Menu
            buttonClass="bg-base-100 hover:bg-base-200 border border-base-400 btn-class text-sm flex font-condensed h-full text-base-600"
            buttonContent={
                <div className="flex items-center text-left px-2">
                    {text}
                    <ChevronDown className="pointer-events-none ml-2" />
                </div>
            }
            options={options}
            className="h-full min-w-32"
        />
    );
};

DashboardMenu.propTypes = {
    text: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired
};

export default DashboardMenu;
