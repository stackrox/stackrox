import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { standardBaseTypes } from 'constants/entityTypes';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import ListTable from './Table';
import SidePanel from './SidePanel';

class ComplianceList extends Component {
    static propTypes = {
        entityType: PropTypes.string.isRequired,
        query: PropTypes.shape({}),
        location: ReactRouterPropTypes.location.isRequired
    };

    static defaultProps = {
        query: null
    };

    constructor(props) {
        super(props);
        this.state = {
            selectedRow: null
        };
    }

    componentWillReceiveProps(nextProps) {
        if (nextProps.location !== this.props.location) {
            this.setState({ selectedRow: null });
        }
    }

    updateSelectedRow = selectedRow => this.setState({ selectedRow });

    clearSelectedRow = () => {
        this.setState({ selectedRow: null });
    };

    render() {
        const { selectedRow } = this.state;
        const { entityType, query } = this.props;

        let sidePanel;
        if (selectedRow) {
            const { name, id: selectedId, control, standardId } = selectedRow;
            const linkText = control ? `${standardBaseTypes[standardId]} ${control}` : name;

            sidePanel = (
                <SidePanel
                    entityType={entityType}
                    entityId={selectedId}
                    clearSelectedRow={this.clearSelectedRow}
                    linkText={linkText}
                    standardId={standardId}
                />
            );
        }

        return (
            <div className="flex flex-1 overflow-y-auto">
                <ListTable
                    selectedRow={selectedRow}
                    entityType={entityType}
                    query={query}
                    updateSelectedRow={this.updateSelectedRow}
                    pdfId="capture-list"
                />
                {sidePanel}
            </div>
        );
    }
}

export default withRouter(ComplianceList);
