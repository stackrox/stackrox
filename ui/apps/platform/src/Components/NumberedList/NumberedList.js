import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { Tooltip } from '@patternfly/react-core';

import DetailedTooltipContent from 'Components/DetailedTooltipContent';

const leftSideClasses = 'p-2 text-sm text-primary-800 w-full';

const NumberedList = ({ data, linkLeftOnly }) => {
    const list = data.map(({ text, subText, url, component, tooltip }, i) => {
        const className = `flex items-center ${i !== 0 ? 'border-t border-base-300' : ''} ${
            url ? 'hover:bg-base-200' : ''
        }`;
        let leftSide = (
            <>
                {i + 1}.&nbsp;{text}&nbsp;
                {subText && <span className="text-base-500">{subText}</span>}
            </>
        );
        if (url && linkLeftOnly) {
            leftSide = (
                <Link
                    data-testid="numbered-list-item-name"
                    className={`${leftSideClasses} no-underline w-full truncate`}
                    to={url}
                >
                    <span className="w-full block truncate">{leftSide}</span>
                </Link>
            );
        } else {
            leftSide = (
                <span
                    data-testid="numbered-list-item-name"
                    className={`${leftSideClasses} truncate`}
                >
                    {leftSide}
                </span>
            );
        }
        let content = (
            <>
                {leftSide}
                <div className="flex justify-end pr-2 whitespace-nowrap items-center">
                    {component}
                </div>
            </>
        );
        if (url && !linkLeftOnly) {
            content = (
                <Link className="flex items-center no-underline w-full" to={url}>
                    {content}
                </Link>
            );
        }
        const contentWrapper = (
            <div className="w-full flex justify-between relative">{content}</div>
        );
        return (
            <li key={text + subText + url} className={className}>
                {tooltip && (
                    <Tooltip
                        isContentLeftAligned
                        content={
                            <DetailedTooltipContent
                                title={tooltip.title}
                                body={tooltip.body}
                                subtitle={tooltip.subtitle}
                                footer={tooltip.footer}
                            />
                        }
                    >
                        {contentWrapper}
                    </Tooltip>
                )}
                {!tooltip && contentWrapper}
            </li>
        );
    });
    return <ul className="leading-loose">{list}</ul>;
};

NumberedList.propTypes = {
    data: PropTypes.arrayOf(
        PropTypes.shape({
            text: PropTypes.string.isRequired,
            subText: PropTypes.string,
            components: PropTypes.element,
            url: PropTypes.string,
        })
    ),
    linkLeftOnly: PropTypes.bool,
};

NumberedList.defaultProps = {
    data: [],
    linkLeftOnly: false,
};

export default NumberedList;
