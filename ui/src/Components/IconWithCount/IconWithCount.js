import React from 'react';
import PropTypes from 'prop-types';

import Loader from 'Components/Loader';

function IconWithCount({ Icon, count, isLoading }) {
    let content = <Loader message={null} />;
    if (!isLoading) {
        content = (
            <>
                <span className="mr-1 text-sm">{count}</span>
                <Icon className="h-3 w-3 text-base-600" />
            </>
        );
    }
    return <span className="flex items-center border-base-300 border-r mr-2 pr-2">{content}</span>;
}

IconWithCount.propTypes = {
    Icon: PropTypes.element.isRequired,
    count: PropTypes.number.isRequired,
    isLoading: PropTypes.bool
};

IconWithCount.defaultProps = {
    isLoading: false
};

export default IconWithCount;
