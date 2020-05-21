import React from 'react';
import PropTypes from 'prop-types';

import CollapsibleCard from 'Components/CollapsibleCard';
import { Creatable } from 'Components/ReactSelect';
import Loader from 'Components/Loader';

const Tags = ({
    title,
    tags,
    onChange,
    isDisabled,
    defaultOpen,
    isCollapsible,
    isLoading,
    autoComplete,
    onInputChange,
}) => {
    const options = autoComplete.map((option) => ({ label: option, value: option }));

    let content = <Loader />;
    if (!isLoading) {
        content = (
            <Creatable
                id="tags"
                name="tags"
                options={options}
                placeholder="No tags created yet. Create new tags."
                onChange={onChange}
                onInputChange={onInputChange}
                className="block w-full bg-base-100 border-base-300 text-base-600 z-1 focus:border-base-500"
                value={tags}
                isDisabled={isDisabled}
                isMulti
                disallowWhitespace
            />
        );
    }

    // if no title is present, just show the input for tags
    if (!title) return content;

    return (
        <CollapsibleCard
            cardClassName="border border-base-400 h-full"
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
    onInputChange: PropTypes.func.isRequired,
    defaultOpen: PropTypes.bool,
    isCollapsible: PropTypes.bool,
    isLoading: PropTypes.bool,
    isDisabled: PropTypes.bool,
    autoComplete: PropTypes.arrayOf(PropTypes.string),
};

Tags.defaultProps = {
    title: null,
    tags: [],
    defaultOpen: false,
    isCollapsible: true,
    isLoading: false,
    isDisabled: false,
    autoComplete: [],
};

export default Tags;
