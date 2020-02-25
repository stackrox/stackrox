import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import CollapsibleCard from 'Components/CollapsibleCard';
import { Creatable } from 'Components/ReactSelect';

function noOptionsMessage() {
    return null;
}

const Tags = ({ className, type, tags, onChange, disabled, defaultOpen, isCollapsible }) => {
    const { length } = tags;
    const options = tags.map(tag => ({ label: tag, value: tag }));

    return (
        <CollapsibleCard
            cardClassName={className}
            title={`${length} ${type} ${pluralize('Tag', length)}`}
            open={defaultOpen}
            isCollapsible={isCollapsible}
        >
            <div className="m-3">
                <Creatable
                    id="tags"
                    name="tags"
                    options={options}
                    placeholder={`No tags created for this ${type}`}
                    onChange={onChange}
                    className="block w-full bg-base-100 border-base-300 text-base-600 z-1 focus:border-base-500"
                    value={tags}
                    disabled={disabled}
                    isMulti
                    noOptionsMessage={noOptionsMessage}
                />
            </div>
        </CollapsibleCard>
    );
};

Tags.propTypes = {
    type: PropTypes.string.isRequired,
    tags: PropTypes.arrayOf(PropTypes.string),
    onChange: PropTypes.func.isRequired,
    disabled: PropTypes.bool,
    defaultOpen: PropTypes.bool,
    isCollapsible: PropTypes.bool,
    className: PropTypes.string
};

Tags.defaultProps = {
    tags: [],
    disabled: false,
    defaultOpen: false,
    isCollapsible: true,
    className: 'border border-base-400'
};

export default Tags;
