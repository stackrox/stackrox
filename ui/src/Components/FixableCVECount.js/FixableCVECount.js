import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const orientations = ['horizontal', 'vertical'];

const getOrientationClassName = orientation => {
    if (orientation === 'vertical') return '';
    return 'flex';
};

const FixableCVECount = ({ cves, fixable, url, orientation }) => {
    const className = `text-base leading-normal ${getOrientationClassName(orientation)}`;
    let content = (
        <div className={className}>
            {!!cves && <div className="text-primary-800 font-600 mx-1">{cves} CVES</div>}
            {!!fixable && <div className="text-success-800 font-600">({fixable} Fixable)</div>}
        </div>
    );
    if (url) content = <Link to={url}>{content}</Link>;
    return content;
};

FixableCVECount.propTypes = {
    cves: PropTypes.number,
    fixable: PropTypes.number,
    url: PropTypes.string,
    orientation: PropTypes.oneOf(orientations)
};

FixableCVECount.defaultProps = {
    cves: 0,
    fixable: 0,
    url: null,
    orientation: 'horizontal'
};

export default FixableCVECount;
