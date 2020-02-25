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

const LabelChip = ({ text, type, size, fade, dataTestId }) => {
    let className =
        'inline-block border rounded font-600 text-center whitespace-no-wrap min-h-6 flex justify-center items-center';
    className = getClassNameBySize(className, size);
    const colorType = colorTypes.find(datum => datum === type) || defaultColorType;
    className = `${className} bg-${colorType}-200 border-${colorType}-400 text-${colorType}-800 capitalize ${
        fade ? 'opacity-50' : ''
    }`;
    return (
        <span className={className} data-testid={dataTestId}>
            <span>{text}</span>
        </span>
    );
};

LabelChip.propTypes = {
    text: PropTypes.string.isRequired,
    type: PropTypes.oneOf(colorTypes),
    size: PropTypes.oneOf(['small', 'medium', 'large']),
    fade: PropTypes.bool,
    dataTestId: PropTypes.string
};

LabelChip.defaultProps = {
    type: defaultColorType,
    size: 'medium',
    fade: false,
    dataTestId: 'label-chip'
};

export default LabelChip;
