import React from 'react';
import PropTypes from 'prop-types';

const DetailedTooltipOverlay = ({ title, subtitle, body, footer }) => {
    if (!title || !body) {
        return null;
    }

    return (
        <div className="rox-tooltip-overlay min-w-32 max-h-100 flex flex-col flex-1">
            <div className="text-left flex flex-col border-b border-primary-400 leading-normal py-1 px-2">
                <h1 className="font-700 text-lg" data-testid="tooltip-title">
                    {title}
                </h1>
                {subtitle && <span data-testid="tooltip-subtitle">{subtitle}</span>}
            </div>
            <div
                className="flex flex-1 flex-col text-left overflow-auto p-2 text-sm"
                data-testid="tooltip-body"
            >
                {body}
            </div>
            {!!footer && (
                <footer
                    className="p-2 font-700 text-left text-sm leading-loose"
                    data-testid="tooltip-footer"
                >
                    {footer}
                </footer>
            )}
        </div>
    );
};

DetailedTooltipOverlay.propTypes = {
    title: PropTypes.string.isRequired,
    body: PropTypes.node.isRequired,
    subtitle: PropTypes.string,
    footer: PropTypes.node,
};

DetailedTooltipOverlay.defaultProps = {
    subtitle: '',
    footer: '',
};

export default DetailedTooltipOverlay;
