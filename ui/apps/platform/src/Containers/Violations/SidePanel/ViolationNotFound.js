import React from 'react';
import { Link } from 'react-router-dom';

import Message from 'Components/Message';

function ViolationNotFound() {
    const message = (
        <div>
            Violation not found. This violation may have been deleted due to &nbsp;
            <Link to="/main/systemconfig" className="text-primary-700">
                data retention settings
            </Link>
        </div>
    );
    return (
        <div className="h-full flex-1 bg-base-200 border-r border-l border-b border-base-400 p-3">
            <Message message={message} type="error" />
        </div>
    );
}

export default ViolationNotFound;
