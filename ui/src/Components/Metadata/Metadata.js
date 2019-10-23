import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import Popper from 'Components/Popper';
import pluralize from 'pluralize';

const ResourceCountPopper = ({ data, label, renderContent, reactOutsideClassName }) => {
    const { length } = data;
    return (
        <Popper
            disabled={!length}
            placement="bottom"
            reactOutsideClassName={reactOutsideClassName}
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
    reactOutsideClassName: PropTypes.string,
    renderContent: PropTypes.func.isRequired
};

ResourceCountPopper.defaultProps = {
    reactOutsideClassName: null
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

const Metadata = ({ keyValuePairs, title, labels, annotations, whitelists, secrets, ...rest }) => {
    const keyValueList = keyValuePairs.map(({ key, value }) => (
        <li className="border-b border-base-300 py-2" key={key}>
            <span className="text-base-700 font-600 mr-2">{key}:</span>
            {value}
        </li>
    ));

    const keyValueClasses = `flex-1 list-reset border-base-300 overflow-hidden px-2 ${
        labels || annotations || whitelists || secrets ? ' border-r' : ''
    }`;

    return (
        <Widget header={title} {...rest}>
            <div className="flex w-full text-sm">
                <ul className={keyValueClasses}>{keyValueList}</ul>
                <ul className="list-reset">
                    {labels && (
                        <li className="m-4">
                            <ResourceCountPopper
                                data={labels}
                                reactOutsideClassName="ignore-label-onclickoutside"
                                label="Label"
                                renderContent={renderKeyValuePairs}
                            />
                        </li>
                    )}
                    {annotations && (
                        <li className="m-4">
                            <ResourceCountPopper
                                data={annotations}
                                reactOutsideClassName="ignore-annotation-onclickoutside"
                                label="Annotation"
                                renderContent={renderKeyValuePairs}
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
    labels: PropTypes.arrayOf(PropTypes.shape({})),
    annotations: PropTypes.arrayOf(PropTypes.shape({})),
    whitelists: PropTypes.arrayOf(PropTypes.shape({})),
    secrets: PropTypes.arrayOf(PropTypes.shape({}))
};

Metadata.defaultProps = {
    title: 'Metadata',
    labels: null,
    annotations: null,
    whitelists: null,
    secrets: null
};

export default Metadata;
