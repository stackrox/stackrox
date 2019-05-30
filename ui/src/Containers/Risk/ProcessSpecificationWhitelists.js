import React from 'react';
import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import PropTypes from 'prop-types';
import ProcessWhitelist from './ProcessWhitelist';

const ProcessSpecificationWhitelists = ({ processesWhitelist }) => (
    <div className="pl-3 pr-3">
        <h3 className="border-b border-base-500 pb-2 mb-3">Spec Container Whitelists</h3>
        <ul className="list-reset border-b border-base-300 leading-normal hover:bg-primary-100 hover:border-primary-300">
            {processesWhitelist.map(({ data }) => (
                <ProcessWhitelist process={data} key={data.key.containerName} />
            ))}
        </ul>
    </div>
);

ProcessSpecificationWhitelists.propTypes = {
    processesWhitelist: PropTypes.arrayOf(PropTypes.object)
};

ProcessSpecificationWhitelists.defaultProps = {
    processesWhitelist: []
};

const mapStateToProps = createStructuredSelector({
    processesWhitelist: selectors.getProcessesWhitelistByDeployment
});

export default connect(
    mapStateToProps,
    null
)(ProcessSpecificationWhitelists);
