import React from 'react';
import PropTypes from 'prop-types';

import ConfigBannerDetailWidget from './ConfigBannerDetailWidget';
import ConfigLoginDetailWidget from './ConfigLoginDetailWidget';
import ConfigDataRetentionDetailWidget from './ConfigDataRetentionDetailWidget';
import { pageLayoutClassName } from './SystemConfig.constants';

const Detail = ({ config }) => (
    <div className={pageLayoutClassName}>
        <div className="px-3 pb-5 w-full">
            <ConfigDataRetentionDetailWidget config={config} />
        </div>
        <div className="flex flex-col justify-between md:flex-row pb-5 w-full">
            <ConfigBannerDetailWidget type="header" config={config} />
            <ConfigBannerDetailWidget type="footer" config={config} />
        </div>
        <div className="px-3 pb-5 w-full">
            <ConfigLoginDetailWidget config={config} />
        </div>
    </div>
);

Detail.propTypes = {
    config: PropTypes.shape({}).isRequired
};

export default Detail;
