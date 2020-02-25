import React, { useState } from 'react';
import PropTypes from 'prop-types';

import Tags from 'Components/Tags';

const defaultTags = ['spicy', 'mild', 'bland'];

const AnalystTags = ({ className, type }) => {
    const [tags, setTags] = useState(defaultTags);

    return <Tags className={className} type={type} tags={tags} onChange={setTags} defaultOpen />;
};

AnalystTags.propTypes = {
    type: PropTypes.string.isRequired,
    className: PropTypes.string
};

AnalystTags.defaultProps = {
    className: 'border border-base-400'
};

export default React.memo(AnalystTags);
