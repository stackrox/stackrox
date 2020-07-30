import PropTypes from 'prop-types';
import React from 'react';
import { AlertCircle, AlertTriangle } from 'react-feather';
import dateFns from 'date-fns';
import Tooltip from '../../Components/Tooltip';
import TooltipOverlay from '../../Components/TooltipOverlay';

const statusTypes = {
    info: {
        color: 'text-base-600',
        icon: null,
    },
    warn: {
        color: 'text-warning-700',
        icon: AlertCircle,
    },
    error: {
        color: 'text-alert-700',
        icon: AlertTriangle,
    },
};

function CredentialExpiration({ showExpiringSoon, type, expiration, diffInWords }) {
    const chosenType = statusTypes[type] || statusTypes.info;

    // make icon element a Dynamic Component Name, so we can inject it into the JSX template
    // see: https://medium.com/@Carmichaelize/dynamic-tag-names-in-react-and-jsx-17e366a684e9
    const IconElement = chosenType.icon || null;

    return (
        <Tooltip content={<TooltipOverlay>{dateFns.format(expiration)}</TooltipOverlay>}>
            <div className={`flex items-center content-center ${chosenType.color}`}>
                {IconElement && <IconElement />}
                <div className="flex flex-col justify-center ml-2">
                    {showExpiringSoon && 'Expiring soon: '}
                    {diffInWords} remaining
                </div>
            </div>
        </Tooltip>
    );
}

CredentialExpiration.propTypes = {
    showExpiringSoon: PropTypes.bool,
    type: PropTypes.string,
    expiration: PropTypes.string.isRequired,
    diffInWords: PropTypes.string.isRequired,
};

CredentialExpiration.defaultProps = {
    showExpiringSoon: false,
    type: 'info',
};

export default CredentialExpiration;
