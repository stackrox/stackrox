import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import Popper from 'Components/Popper';
import pluralize from 'pluralize';

const ResourceCountPopper = ({ data, label, renderContent }) => {
    const { length } = data;
    return (
        <Popper
            disabled={!length}
            placement="bottom"
            buttonClass={`rounded border border-base-400 p-1 px-4 text-center text-sm ${length &&
                'hover:bg-base-200'}`}
            buttonContent={
                <div>
                    {length} {pluralize(label, length)}
                </div>
            }
            popperContent={
                <div className="border border-base-300 p-4 shadow bg-base-100 whitespace-no-wrap">
                    {renderContent(data)}
                </div>
            }
        />
    );
};

ResourceCountPopper.propTypes = {
    data: PropTypes.arrayOf(PropTypes.shape()).isRequired,
    label: PropTypes.string.isRequired,
    renderContent: PropTypes.func.isRequired
};

const renderKeyValuePairs = data => {
    return data.map(({ key, value }) => (
        <div className="mt-2" key={key}>
            {key} : {value}
        </div>
    ));
};
const renderName = data => {
    return data.map(({ name }) => (
        <div className="mt-2" key={name}>
            {name}
        </div>
    ));
};

const Metadata = ({ keyValuePairs, labels, annotations, whitelists, secrets, ...rest }) => {
    const keyValueList = keyValuePairs.map(({ key, value }) => (
        <li className="border-b border-base-300 px-4 py-2" key={key}>
            <span className="text-base-700 font-600 mr-2">{key}:</span>
            {value}
        </li>
    ));
    return (
        <Widget header="Metadata" {...rest}>
            <div className="flex w-full text-sm">
                <ul className="flex-1 list-reset border-r border-base-300">{keyValueList}</ul>
                <ul className="list-reset">
                    {labels && (
                        <li className="m-4">
                            <ResourceCountPopper
                                data={labels}
                                label="Label"
                                renderContent={renderKeyValuePairs}
                            />
                        </li>
                    )}
                    {annotations && (
                        <li className="m-4">
                            <ResourceCountPopper
                                data={annotations}
                                label="Annotation"
                                renderContent={renderKeyValuePairs}
                            />
                        </li>
                    )}
                    {whitelists && (
                        <li className="m-4">
                            <ResourceCountPopper
                                data={whitelists}
                                label="Whitelist"
                                renderContent={renderName}
                            />
                        </li>
                    )}
                    {secrets && (
                        <li className="m-4">
                            <ResourceCountPopper
                                data={secrets}
                                label="Image Pull Secret"
                                renderContent={renderName}
                            />
                        </li>
                    )}
                </ul>
            </div>
        </Widget>
    );
};

PropTypes.propTypes = {
    keyValuePairs: PropTypes.arrayOf(
        PropTypes.shape({
            key: PropTypes.string.isRequired,
            value: PropTypes.oneOf([PropTypes.string.isRequired, PropTypes.element.isRequired])
        })
    )
};

export default Metadata;
