import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import CollapsibleCard from 'Components/CollapsibleCard';
import { Creatable } from 'Components/ReactSelect';
import Loader from 'Components/Loader';

function noOptionsMessage() {
    return null;
}

const Tags = ({
    className,
    label,
    tags,
    onChange,
    isDisabled,
    defaultOpen,
    isCollapsible,
    isLoading
}) => {
    const { length } = tags;
    const options = tags.map(tag => ({ label: tag, value: tag }));

    let content = <Loader />;
    if (!isLoading) {
        content = (
            <Creatable
                id="tags"
                name="tags"
                options={options}
                placeholder={`No ${label} tags were created.`}
                onChange={onChange}
                className="block w-full bg-base-100 border-base-300 text-base-600 z-1 focus:border-base-500"
                value={tags}
                isDisabled={isDisabled}
                isMulti
                noOptionsMessage={noOptionsMessage}
            />
        );
    }

    return (
        <CollapsibleCard
            cardClassName={className}
            title={`${length} ${label} ${pluralize('Tag', length)}`}
            open={defaultOpen}
            isCollapsible={isCollapsible}
        >
            <div className="m-3">{content}</div>
        </CollapsibleCard>
    );
};

Tags.propTypes = {
    label: PropTypes.string.isRequired,
    tags: PropTypes.arrayOf(PropTypes.string),
    onChange: PropTypes.func.isRequired,
    defaultOpen: PropTypes.bool,
    isCollapsible: PropTypes.bool,
    className: PropTypes.string,
    isLoading: PropTypes.bool,
    isDisabled: PropTypes.bool
};

Tags.defaultProps = {
    tags: [],
    defaultOpen: false,
    isCollapsible: true,
    className: 'border border-base-400',
    isLoading: false,
    isDisabled: false
};

export default Tags;
