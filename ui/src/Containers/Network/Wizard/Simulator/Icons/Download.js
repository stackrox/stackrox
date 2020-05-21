import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';
import { selectors } from 'reducers';
import download from 'utils/download';

class Download extends Component {
    static propTypes = {
        modificationName: PropTypes.string.isRequired,
        modification: PropTypes.shape({
            applyYaml: PropTypes.string.isRequired,
        }).isRequired,
    };

    onClick = () => {
        const { applyYaml } = this.props.modification;
        const formattedYaml = applyYaml.split('\\n').join('\n');

        const { modificationName } = this.props;
        const yamlName = modificationName.split(/.yaml|.yml/g)[0];
        download(`${yamlName}.yaml`, formattedYaml, 'yaml');
    };

    render() {
        return (
            <Tooltip content={<TooltipOverlay>Download YAML</TooltipOverlay>}>
                <button
                    type="button"
                    className="inline-block px-2 py-2 border-base-300 cursor-pointer"
                    onClick={this.onClick}
                >
                    <Icon.Download className="h-4 w-4 text-base-500 hover:bg-base-200" />
                </button>
            </Tooltip>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    modificationName: selectors.getNetworkPolicyModificationName,
    modification: selectors.getNetworkPolicyModification,
});

export default connect(mapStateToProps)(Download);
