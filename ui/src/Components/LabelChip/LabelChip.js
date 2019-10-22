import React from 'react';
import PropTypes from 'prop-types';
import { colorTypes, defaultColorType } from 'constants/visuals/colors';

const getClassNameBySize = (className, size) => {
    let sizeClassName = '';
    switch (size) {
        case 'small':
            sizeClassName = 'text-xs px-1';
            break;
        case 'large':
            sizeClassName = 'text-base px-2 py-1';
            break;
        case 'medium':
        default:
            sizeClassName = 'text-base px-2';
            break;
    }
    return `${className} ${sizeClassName}`;
};

const LabelChip = ({ text, type, size }) => {
    let className = 'inline-block border rounded font-600 text-center';
    className = getClassNameBySize(className, size);
    const colorType = colorTypes.find(datum => datum === type) || defaultColorType;
    className = `${className} bg-${colorType}-200 border-${colorType}-400 text-${colorType}-800`;
    return <span className={className}>{text}</span>;
};

LabelChip.propTypes = {
    text: PropTypes.string.isRequired,
    type: PropTypes.oneOf(colorTypes),
    size: PropTypes.oneOf(['small', 'medium', 'large'])
};

LabelChip.defaultProps = {
    type: defaultColorType,
    size: 'medium'
};

export default LabelChip;
