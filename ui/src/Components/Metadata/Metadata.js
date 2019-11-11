import React from 'react';
import PropTypes from 'prop-types';

import MetadataStatsList from 'Components/MetadataStatsList';
import Widget from 'Components/Widget';
import ResourceCountPopper from 'Components/ResourceCountPopper';

const renderName = data => {
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
    whitelists,
    secrets,
    ...rest
}) => {
    const keyValueList = keyValuePairs.map(({ key, value }) => (
        <li className="border-b border-base-300 py-3" key={key}>
            <span className="text-base-600 font-700 mr-2">{key}:</span>
            <span className="font-600">{value}</span>
        </li>
    ));

    const keyValueClasses = `flex-1 list-reset border-base-300 overflow-hidden px-3 ${
        labels || annotations || whitelists || secrets ? ' border-r' : ''
    }`;

    return (
        <Widget header={title} {...rest}>
            <div className="flex flex-col w-full">
                {statTiles && statTiles.length > 0 && <MetadataStatsList statTiles={statTiles} />}
                <div className="flex w-full h-full">
                    <ul className={keyValueClasses}>{keyValueList}</ul>
                    <ul className="list-reset">
                        {labels && (
                            <li className="m-4">
                                <ResourceCountPopper
                                    data={labels}
                                    reactOutsideClassName="ignore-label-onclickoutside"
                                    label="Label"
                                />
                            </li>
                        )}
                        {annotations && (
                            <li className="m-4">
                                <ResourceCountPopper
                                    data={annotations}
                                    reactOutsideClassName="ignore-annotation-onclickoutside"
                                    label="Annotation"
                                />
                            </li>
                        )}
                        {whitelists && (
                            <li className="m-4">
                                <ResourceCountPopper
                                    data={whitelists}
                                    reactOutsideClassName="ignore-whitelist-onclickoutside"
                                    label="Whitelist"
                                    renderContent={renderName}
                                />
                            </li>
                        )}
                        {secrets && (
                            <li className="m-4">
                                <ResourceCountPopper
                                    data={secrets}
                                    reactOutsideClassName="ignore-secret-onclickoutside"
                                    label="Image Pull Secret"
                                    renderContent={renderName}
                                />
                            </li>
                        )}
                    </ul>
                </div>
            </div>
        </Widget>
    );
};

Metadata.propTypes = {
    keyValuePairs: PropTypes.arrayOf(
        PropTypes.shape({
            key: PropTypes.string.isRequired,
            value: PropTypes.oneOfType([PropTypes.string, PropTypes.element, PropTypes.number])
        })
    ).isRequired,
    title: PropTypes.string,
    statTiles: PropTypes.arrayOf(PropTypes.node),
    labels: PropTypes.arrayOf(PropTypes.shape({})),
    annotations: PropTypes.arrayOf(PropTypes.shape({})),
    whitelists: PropTypes.arrayOf(PropTypes.shape({})),
    secrets: PropTypes.arrayOf(PropTypes.shape({}))
};

Metadata.defaultProps = {
    title: 'Metadata',
    statTiles: null,
    labels: null,
    annotations: null,
    whitelists: null,
    secrets: null
};

export default Metadata;
