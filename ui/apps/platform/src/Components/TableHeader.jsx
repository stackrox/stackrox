import React from 'react';
import PropTypes from 'prop-types';
import { Tooltip } from '@patternfly/react-core';

const maxSize = 1000;

const TableHeader = (props) => {
    let headerText = '';
    if (props.selectionCount !== 0) {
        headerText = `${props.selectionCount} ${props.type}${
            props.selectionCount === 1 ? '' : 's'
        } Selected`;
    } else {
        headerText = `${props.length}${props.length === maxSize ? '+' : ''} ${props.type}${
            props.length === 1 ? '' : 's'
        } ${props.isViewFiltered ? 'Matched' : ''} ${
            props.length === maxSize ? 'are available' : ''
        }
    `;
    }
    let component = (
        <div
            className="overflow-hidden mx-4 flex text-base-600 items-center leading-normal font-700"
            data-testid="filtered-header"
        >
            <Tooltip content={headerText}>
                <div className="truncate flex-none">{headerText}</div>
            </Tooltip>
        </div>
    );

    if (props.length >= maxSize) {
        component = (
            <div className="pt-2">
                {component}
                <div className="pl-4 opacity-75">
                    Please add a filter to narrow down your results.
                </div>
            </div>
        );
    }
    return component;
};

TableHeader.propTypes = {
    length: PropTypes.number.isRequired,
    selectionCount: PropTypes.number,
    type: PropTypes.string.isRequired,
    isViewFiltered: PropTypes.bool,
};

TableHeader.defaultProps = {
    isViewFiltered: false,
    selectionCount: 0,
};

export default TableHeader;
