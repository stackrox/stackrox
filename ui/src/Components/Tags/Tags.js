import React from 'react';
import PropTypes from 'prop-types';

import CollapsibleCard from 'Components/CollapsibleCard';
import { Creatable } from 'Components/ReactSelect';
import Loader from 'Components/Loader';

function noOptionsMessage() {
    return null;
}

const Tags = ({
    className,
    title,
    tags,
    onChange,
    isDisabled,
    defaultOpen,
    isCollapsible,
    isLoading
}) => {
    const options = tags.map(tag => ({ label: tag, value: tag }));

    let content = <Loader />;
    if (!isLoading) {
        content = (
            <Creatable
                id="tags"
                name="tags"
                options={options}
                placeholder="No tags created yet. Create new tags."
                onChange={onChange}
                className="block w-full bg-base-100 border-base-300 text-base-600 z-1 focus:border-base-500"
                value={tags}
                isDisabled={isDisabled}
                isMulti
                noOptionsMessage={noOptionsMessage}
            />
        );
    }

    // if no title is present, just show the input for tags
    if (!title) return content;

    return (
        <CollapsibleCard
            cardClassName={className}
            title={title}
            open={defaultOpen}
            isCollapsible={isCollapsible}
        >
            <div className="m-3">{content}</div>
        </CollapsibleCard>
    );
};

Tags.propTypes = {
    title: PropTypes.string,
    tags: PropTypes.arrayOf(PropTypes.string),
    onChange: PropTypes.func.isRequired,
    defaultOpen: PropTypes.bool,
    isCollapsible: PropTypes.bool,
    className: PropTypes.string,
    isLoading: PropTypes.bool,
    isDisabled: PropTypes.bool
};

Tags.defaultProps = {
    title: null,
    tags: [],
    defaultOpen: false,
    isCollapsible: true,
    className: 'border border-base-400',
    isLoading: false,
    isDisabled: false
};

export default Tags;
