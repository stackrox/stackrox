import React, { Component, ElementType } from 'react';
import Loader from 'Components/Loader';

type Props = Record<string, never>;
type State = {
    component: ElementType | null;
};

export default function asyncComponent(importComponent) {
    class AsyncComponent extends Component<Props, State> {
        isComponentMounted: boolean;

        constructor(props) {
            super(props);
            this.state = {
                component: null,
            };
            this.isComponentMounted = false;
        }

        async componentDidMount() {
            this.isComponentMounted = true;
            const { default: component } = await importComponent();
            if (this.isComponentMounted) {
                this.setState({ component });
            }
        }

        componentWillUnmount() {
            this.isComponentMounted = false;
        }

        render() {
            const C = this.state.component;
            return C ? <C {...this.props} /> : <Loader />;
        }
    }

    return AsyncComponent;
}
