import React from 'react';
import PropTypes from 'prop-types';

import ConfigDetailWidget from './ConfigDetailWidget';

const Detail = ({ config }) => (
    <div className="flex flex-col justify-between md:flex-row overflow-auto px-2 w-full">
        <ConfigDetailWidget type="header" config={config} />
        <ConfigDetailWidget type="footer" config={config} />
    </div>
);

Detail.propTypes = {
    config: PropTypes.shape({}).isRequired
};

export default Detail;
