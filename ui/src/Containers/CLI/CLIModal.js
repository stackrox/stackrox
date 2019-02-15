import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { actions } from 'reducers/cli';
import onClickOutside from 'react-onclickoutside';

const downloadBtnClass =
    'w-48 bg-base-100 px-5 py-3 text-left font-600 uppercase text-sm border-b border-base-300 hover:bg-base-400 hover:text-base-700';

class CLIModal extends Component {
    static propTypes = {
        onClose: PropTypes.func.isRequired,
        onDownload: PropTypes.func.isRequired
    };

    componentDidMount() {
        document.addEventListener('keydown', this.handleKeyDown);
    }

    componentWillUnmount() {
        document.removeEventListener('keydown', this.handleKeyDown);
    }

    handleKeyDown = event => {
        // 'escape' key maps to keycode '27'
        if (event.keyCode === 27) {
            this.props.onClose();
        }
    };

    handleClickOutside = () => {
        this.props.onClose();
    };

    handleDownloadMac = () => {
        this.props.onDownload('darwin');
    };

    handleDownloadLinux = () => {
        this.props.onDownload('linux');
    };

    handleDownloadWindows = () => {
        this.props.onDownload('windows');
    };

    render() {
        return (
            <div className=" pointer-events-none">
                <div className="pointer-events-all items-end flex flex-col">
                    <button
                        onClick={this.handleDownloadMac}
                        type="button"
                        className={downloadBtnClass}
                    >
                        Mac 64-bit
                    </button>
                    <button
                        onClick={this.handleDownloadLinux}
                        type="button"
                        className={downloadBtnClass}
                    >
                        Linux 64-bit
                    </button>
                    <button
                        onClick={this.handleDownloadWindows}
                        type="button"
                        className={downloadBtnClass}
                    >
                        Windows 64-bit
                    </button>
                </div>
            </div>
        );
    }
}

const CLIModalContainer = props => {
    const EnhancedCLIModal = onClickOutside(CLIModal);
    return (
        <div className="search-modal pl-4 pr-4 border-t border-base-300 w-full z-60 absolute">
            <EnhancedCLIModal outsideClickIgnoreClass="ignore-cli-clickoutside" {...props} />
        </div>
    );
};

const mapStateToProps = createStructuredSelector({});

const mapDispatchToProps = {
    onDownload: actions.downloadCLI
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(CLIModalContainer);
