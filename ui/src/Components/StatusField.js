import React from 'react';
import PropTypes from 'prop-types';
import {
    AlertCircle,
    AlertTriangle,
    CheckCircle,
    DownloadCloud,
    Info,
    Loader
} from 'react-feather';

const statusTypes = {
    info: {
        color: 'text-base-600',
        icon: Info
    },
    current: {
        color: 'text-base-600',
        icon: CheckCircle
    },
    download: {
        color: 'text-tertiary-700',
        icon: DownloadCloud
    },
    progress: {
        color: 'text-success-600',
        icon: Loader
    },
    failure: {
        color: 'text-alert-700',
        icon: AlertTriangle
    },
    intervention: {
        color: 'text-warning-700',
        icon: AlertCircle
    }
};

function StatusField({ displayValue, type, action }) {
    const chosenType = statusTypes[type] || statusTypes.info;

    // make icon element a Dynamic Component Name, so we can inject it into the JSX template
    // see: https://medium.com/@Carmichaelize/dynamic-tag-names-in-react-and-jsx-17e366a684e9
    const IconElement = chosenType.icon;

    return (
        <div className={`flex items-center content-center ${chosenType.color}`}>
            <IconElement />
            <div className="flex flex-col justify-center ml-2">
                <div>{displayValue}</div>
                {action !== null && (
                    <button
                        type="button"
                        className="bg-transparent underline font-semibold p-0 m-0"
                        onClick={action.actionHandler}
                    >
                        {action.actionText}
                    </button>
                )}
            </div>
        </div>
    );
}

StatusField.propTypes = {
    displayValue: PropTypes.string.isRequired,
    type: PropTypes.string,
    action: PropTypes.shape({
        actionHandlerText: PropTypes.string.isRequired,
        actionHandler: PropTypes.func.isRequired
    })
};

StatusField.defaultProps = {
    type: 'info',
    action: null
};

export default StatusField;
