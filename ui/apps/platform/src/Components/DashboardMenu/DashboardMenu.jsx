import React from 'react';
import PropTypes from 'prop-types';

import Menu from 'Components/Menu';

const DashboardMenu = ({ text, options }) => {
    return (
        <Menu
            buttonClass="bg-base-100 hover:bg-base-200 border border-base-400 btn-class flex h-full text-base-600"
            buttonText={text}
            options={options}
            className="h-full min-w-32"
        />
    );
};

DashboardMenu.propTypes = {
    text: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
};

export default DashboardMenu;
