import React from 'react';
import PropTypes from 'prop-types';

const HeaderWithSubText = ({ header, subText }) => {
    return (
        <div className="flex flex-col font-700 items-center items-stretch justify-center leading-normal px-4 text-base-600">
            <div className="font-700 text-base-600" data-testid="header">
                {header}
            </div>
            <div className="text-base-500 text-xs font-700" data-testid="subText">
                {subText}
            </div>
        </div>
    );
};

HeaderWithSubText.propTypes = {
    header: PropTypes.string.isRequired,
    subText: PropTypes.string.isRequired,
};

export default HeaderWithSubText;
