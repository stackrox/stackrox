import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { Tooltip } from 'react-tippy';
import { actions as processActions } from 'reducers/processes';
import CollapsibleCard from 'Components/CollapsibleCard';

const titleClassName =
    'border-b border-base-300 leading-normal cursor-pointer flex justify-between h-14';
const suspiciousProcessClassName = 'bg-alert-200 hover:bg-alert-200 hover:border-alert-300';
const headerClassName = 'bg-base-100 hover:bg-primary-200 hover:border-primary-300';

class ProcessesDiscoveryCard extends Component {
    static propTypes = {
        deploymentId: PropTypes.string.isRequired,
        process: PropTypes.shape({
            name: PropTypes.string.isRequired,
            containerName: PropTypes.string.isRequired,
            suspicious: PropTypes.bool.isRequired,
            groups: PropTypes.arrayOf(PropTypes.shape({})).isRequired
        }).isRequired,
        children: PropTypes.node.isRequired,
        addProcesses: PropTypes.func.isRequired
    };

    addWhitelist = evt => {
        evt.stopPropagation();
        const { name, containerName } = this.props.process;
        const addProcessesQuery = {
            keys: [{ deploymentId: this.props.deploymentId, containerName }],
            addElements: [{ processName: name }]
        };
        this.props.addProcesses(addProcessesQuery);
    };

    renderHeader = (backgroundClass, icon) => {
        const { name, containerName, suspicious } = this.props.process;
        const textClass = suspicious ? 'text-alert-800' : 'text-primary-800';
        return (
            <div className={`${titleClassName} ${backgroundClass}`}>
                <div className={`p-3 ${textClass} flex flex-col`}>
                    <h1 className="text-lg font-700">{name}</h1>
                    <h2 className="text-sm font-600 italic">{`in container ${containerName} `}</h2>
                </div>
                <div className="flex content-center">
                    {suspicious && (
                        <div className="border-l border-r flex items-center justify-center w-16 border-alert-300">
                            <Tooltip
                                useContext
                                position="top"
                                trigger="mouseenter"
                                animation="none"
                                duration={0}
                                arrow
                                html={<span className="text-sm">Add to whitelist</span>}
                                unmountHTMLWhenHide
                            >
                                <button
                                    type="button"
                                    onClick={this.addWhitelist}
                                    className="border rounded p-px mr-3 ml-3 border-alert-800 flex items-center hover:bg-alert-300"
                                >
                                    <Icon.Plus className="h-4 w-4 text-alert-800" />
                                </button>
                            </Tooltip>
                        </div>
                    )}
                    <button type="button" className={`pl-3 pr-3 ${suspicious && 'text-alert-800'}`}>
                        {icon}
                    </button>
                </div>
            </div>
        );
    };

    renderWhenOpened = () =>
        this.renderHeader(
            this.props.process.suspicious ? suspiciousProcessClassName : headerClassName,
            <Icon.ChevronUp className="h-4 w-4" />
        );

    renderWhenClosed = () =>
        this.renderHeader(
            this.props.process.suspicious ? suspiciousProcessClassName : headerClassName,
            <Icon.ChevronDown className="h-4 w-4" />
        );

    render() {
        return (
            <CollapsibleCard
                title={this.props.process.name}
                open={false}
                renderWhenOpened={this.renderWhenOpened}
                renderWhenClosed={this.renderWhenClosed}
                cardClassName="border border-base-400"
            >
                {this.props.children}
            </CollapsibleCard>
        );
    }
}

const mapDispatchToProps = {
    addProcesses: processActions.addDeleteProcesses
};

export default connect(
    null,
    mapDispatchToProps
)(ProcessesDiscoveryCard);
