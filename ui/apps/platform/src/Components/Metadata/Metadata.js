import React from 'react';
import PropTypes from 'prop-types';

import MetadataStatsList from 'Components/MetadataStatsList';
import Widget from 'Components/Widget';
import ResourceCountPopper from 'Components/ResourceCountPopper';

const renderName = (data) => {
    return data.map(({ name }) => (
        <div className="mt-2" key={name}>
            {name}
        </div>
    ));
};

const Metadata = ({
    keyValuePairs,
    title,
    statTiles,
    labels,
    annotations,
    exclusions,
    secrets,
    description,
    className,
    ...rest
}) => {
    const keyValueList = keyValuePairs.map(({ key, value }) => (
        <li className="flex border-b border-base-300 py-3" key={key}>
            <span className="text-base-600 font-700 mr-2">{key}:</span>
            <span className="min-w-0" data-testid={`${key}-value`}>
                {value}
            </span>
        </li>
    ));

    const keyValueClasses = `flex-1 last:border-0 border-base-300 overflow-hidden px-3 ${
        labels || annotations || exclusions || secrets ? ' border-r' : ''
    }`;

    return (
        <Widget header={title} className={className} {...rest}>
            <div className="flex flex-col w-full">
                {statTiles && statTiles.length > 0 && <MetadataStatsList statTiles={statTiles} />}
                <div className="flex">
                    <ul className={keyValueClasses}>{keyValueList}</ul>
                    <ul>
                        {labels && (
                            <li className="m-4">
                                <ResourceCountPopper
                                    data={labels}
                                    reactOutsideClassName="ignore-react-onclickoutside"
                                    label="Label"
                                />
                            </li>
                        )}
                        {annotations && (
                            <li className="m-4">
                                <ResourceCountPopper
                                    data={annotations}
                                    reactOutsideClassName="ignore-react-onclickoutside"
                                    label="Annotation"
                                />
                            </li>
                        )}
                        {exclusions && (
                            <li className="m-4">
                                <ResourceCountPopper
                                    data={exclusions}
                                    reactOutsideClassName="ignore-react-onclickoutside"
                                    label="Excluded Scopes"
                                    renderContent={renderName}
                                />
                            </li>
                        )}
                        {secrets && (
                            <li className="m-4">
                                <ResourceCountPopper
                                    data={secrets}
                                    reactOutsideClassName="ignore-react-onclickoutside"
                                    label="Image Pull Secret"
                                    renderContent={renderName}
                                />
                            </li>
                        )}
                    </ul>
                </div>
                {description && (
                    <div className="p-4" data-testid="metadata-description">
                        {description}
                    </div>
                )}
            </div>
        </Widget>
    );
};

Metadata.propTypes = {
    keyValuePairs: PropTypes.arrayOf(
        PropTypes.shape({
            key: PropTypes.string.isRequired,
            value: PropTypes.oneOfType([PropTypes.string, PropTypes.node, PropTypes.number]),
        })
    ).isRequired,
    title: PropTypes.string,
    statTiles: PropTypes.arrayOf(PropTypes.node),
    labels: PropTypes.arrayOf(PropTypes.shape({})),
    annotations: PropTypes.arrayOf(PropTypes.shape({})),
    exclusions: PropTypes.arrayOf(PropTypes.shape({})),
    secrets: PropTypes.arrayOf(PropTypes.shape({})),
    description: PropTypes.string,
    className: PropTypes.string,
};

Metadata.defaultProps = {
    title: 'Metadata',
    statTiles: null,
    labels: null,
    annotations: null,
    exclusions: null,
    secrets: null,
    description: null,
    className: '',
};

export default Metadata;
