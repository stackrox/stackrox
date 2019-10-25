import React from 'react';
import PropTypes from 'prop-types';
import Popper from 'Components/Popper';
import pluralize from 'pluralize';

const renderKeyValuePairs = data => {
    return data.map(({ key, value }) => (
        <div className="mt-2" key={key}>
            {key} : {value}
        </div>
    ));
};

const ResourceCountPopper = ({ data, label, renderContent, reactOutsideClassName }) => {
    const { length } = data;
    return (
        <Popper
            disabled={!length}
            placement="bottom"
            reactOutsideClassName={reactOutsideClassName}
            buttonClass={`rounded border border-base-400 p-1 px-4 text-center text-sm ${length &&
                'hover:bg-base-200'}`}
            buttonContent={`${length} ${pluralize(label, length)}`}
            popperContent={
                <div className="border border-base-300 p-4 shadow bg-base-100 whitespace-no-wrap">
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
            value: PropTypes.oneOf([PropTypes.string, PropTypes.number, PropTypes.bool]).isRequired
        })
    ).isRequired,
    label: PropTypes.string.isRequired,
    reactOutsideClassName: PropTypes.string,
    renderContent: PropTypes.func
};

ResourceCountPopper.defaultProps = {
    reactOutsideClassName: null,
    renderContent: renderKeyValuePairs
};

export default ResourceCountPopper;
