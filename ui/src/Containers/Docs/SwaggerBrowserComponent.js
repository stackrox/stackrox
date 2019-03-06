import React, { Component } from 'react';
import { RedocStandalone } from 'redoc';
import LoadingSection from 'Components/LoadingSection';
import PropTypes from 'prop-types';

import axios from 'services/instance';

class SwaggerBrowserComponent extends Component {
    static propTypes = {
        uri: PropTypes.string.isRequired
    };

    constructor(props) {
        super(props);
        this.state = { loading: true };
    }

    componentDidMount() {
        this.fetchData(this.props.uri);
    }

    componentDidUpdate(prevProps) {
        if (this.props.uri !== prevProps.uri) {
            this.fetchData(this.props.uri);
        }
    }

    fetchData = uri => {
        this.setState({ loading: true });
        axios.get(uri).then(
            response => {
                this.setState({ loading: false, swagger: response.data });
            },
            () => {
                this.setState({ loading: false, error: 'Unable to load API data' });
            }
        );
    };

    render() {
        if (this.state.loading) {
            return <LoadingSection />;
        }
        if (this.state.error) {
            return <div>{this.state.error}</div>;
        }
        if (this.state.swagger) {
            return <RedocStandalone spec={this.state.swagger} />;
        }
        return <div />;
    }
}

export default SwaggerBrowserComponent;
