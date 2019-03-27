import ReduxTextField from 'Components/forms/ReduxTextField';
import React from 'react';
import * as Icon from 'react-feather';
import PropTypes from 'prop-types';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { connect } from 'react-redux';

const removeFieldHandler = (fields, index) => () => {
    fields.remove(index);
};

const addFieldHandler = fields => () => {
    fields.push({});
};

const renderScopes = ({ fields, clusters }) => {
    const clusterOptions = [{ label: 'Cluster', value: '' }].concat(
        clusters.map(cluster => ({
            label: cluster.name,
            value: cluster.id
        }))
    );
    return (
        <div className="w-full">
            <div className="w-full text-right">
                <button className="text-base-500" onClick={addFieldHandler(fields)} type="button">
                    <Icon.PlusSquare size="40" />
                </button>
            </div>
            {fields.map((pair, index) => (
                <div key={pair} className="w-full pb-2">
                    <ReduxSelectField
                        key={`${pair}.cluster`}
                        name={`${pair}.cluster`}
                        component="input"
                        options={clusterOptions}
                        type="text"
                        className="border-2 rounded p-2 my-1 mr-1 border-base-300 w-1/2 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                        placeholder="Cluster"
                    />
                    <ReduxTextField
                        key={`${pair}.namespace`}
                        name={`${pair}.namespace`}
                        component="input"
                        type="text"
                        className="border-2 rounded p-2 my-1 border-base-300 w-1/2 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                        placeholder="Namespace"
                    />
                    <div className="flex">
                        <ReduxTextField
                            key={`${pair}.label.key`}
                            name={`${pair}.label.key`}
                            component="input"
                            type="text"
                            className="border-2 rounded p-2 my-1 mr-1 border-base-300 w-1/2 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                            placeholder="Label Key"
                        />
                        <ReduxTextField
                            key={`${pair}.label.value`}
                            name={`${pair}.label.value`}
                            component="input"
                            type="text"
                            className="border-2 rounded p-2 my-1 border-base-300 w-1/2 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                            placeholder="Label Value"
                        />
                        <button
                            className="ml-2 p-2 my-1 flex rounded-r-sm text-base-100 uppercase text-center text-alert-700 hover:text-alert-800 bg-alert-200 hover:bg-alert-300 border-2 border-alert-300 items-center rounded"
                            onClick={removeFieldHandler(fields, index)}
                            type="button"
                        >
                            <Icon.X size="20" />
                        </button>
                    </div>
                </div>
            ))}
        </div>
    );
};

renderScopes.propTypes = {
    fields: PropTypes.shape({}).isRequired,
    clusters: PropTypes.arrayOf(PropTypes.shape({})).isRequired
};

const mapStateToProps = createStructuredSelector({
    clusters: selectors.getClusters
});

export default connect(
    mapStateToProps,
    {}
)(renderScopes);
