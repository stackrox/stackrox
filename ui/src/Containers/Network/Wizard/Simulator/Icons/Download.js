import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';
import * as Icon from 'react-feather';
import { selectors } from 'reducers';
import download from 'utils/download';

class Download extends Component {
    static propTypes = {
        modificationName: PropTypes.string.isRequired,
        modification: PropTypes.shape({
            applyYaml: PropTypes.string.isRequired
        }).isRequired
    };

    onClick = () => {
        const { applyYaml } = this.props.modification;
        const formattedYaml = applyYaml.split('\\n').join('\n');

        const { modificationName } = this.props;
        download(`${modificationName}.yaml`, formattedYaml, 'yaml');
    };

    render() {
        return (
            <button
                type="button"
                className="inline-block px-2 py-2 border-base-300 cursor-pointer"
                onClick={this.onClick}
            >
                <Tooltip
                    placement="left"
                    overlay={<div>Download YAML</div>}
                    mouseEnterDelay={0.5}
                    mouseLeaveDelay={0}
                >
                    <Icon.Download className="h-4 w-4 text-base-500 hover:bg-base-200" />
                </Tooltip>
            </button>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    modificationName: selectors.getNetworkPolicyModificationName,
    modification: selectors.getNetworkPolicyModification
});

export default connect(mapStateToProps)(Download);
