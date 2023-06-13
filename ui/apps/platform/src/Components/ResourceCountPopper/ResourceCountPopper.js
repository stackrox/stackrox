import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import { ArrowRightCircle } from 'react-feather';

import Popper from 'Components/Popper';

const renderKeyValuePairs = (data) => {
    return data.map(({ key, value }) => (
        <div className="mt-2" key={key}>
            {key} : {value}
        </div>
    ));
};

const ResourceCountPopper = ({ data, label, renderContent, reactOutsideClassName }) => {
    const { length } = data;
    const buttonContent = (
        <div className="flex justify-between items-center">
            <span className="pr-2">{`${length} ${pluralize(label, length)}`}</span>
            <ArrowRightCircle size={12} />
        </div>
    );
    return (
        <Popper
            disabled={!length}
            placement="bottom"
            reactOutsideClassName={reactOutsideClassName}
            buttonClass={`w-full rounded border border-base-400 bg-base-100 p-1 px-2 text-left text-sm ${
                length && 'hover:bg-base-200'
            }`}
            buttonContent={buttonContent}
            popperContent={
                <div className="border border-base-300 p-4 shadow bg-base-100 whitespace-nowrap">
                    {renderContent(data)}
                </div>
            }
        />
    );
};

ResourceCountPopper.propTypes = {
    data: PropTypes.arrayOf(
        PropTypes.shape({
            key: PropTypes.string.isRequired,
            value: PropTypes.oneOfType([PropTypes.string, PropTypes.number, PropTypes.bool])
                .isRequired,
        })
    ).isRequired,
    label: PropTypes.string.isRequired,
    reactOutsideClassName: PropTypes.string,
    renderContent: PropTypes.func,
};

ResourceCountPopper.defaultProps = {
    reactOutsideClassName: null,
    renderContent: renderKeyValuePairs,
};

export default ResourceCountPopper;
